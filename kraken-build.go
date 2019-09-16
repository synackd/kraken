/* kraken-build.go: builds kraken binaries based on a YAML specification
 *
 * Author: J. Lowell Wofford <lowell@lanl.gov>
 *
 * This software is open source software available under the BSD-3 license.
 * Copyright (c) 2018, Triad National Security, LLC
 * See LICENSE file for details.
 */

package main

import (
	"bufio"
	"flag"
	"fmt"
	"go/build"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"

	cp "github.com/otiai10/copy"
	yaml "gopkg.in/yaml.v2"
)

const KrModStr string = "module github.com/hpc/kraken"

// globals (set by flags)
var (
	cfgFile   = flag.String("config", "config/kraken.yaml", "specify the build configuration YAML file")
	buildDir  = flag.String("dir", "build", "specify directory to put built binaries in")
	noCleanup = flag.Bool("noclean", false, "don't cleanup temp dir after build")
	force     = flag.Bool("force", false, "force will overwrite existing build targets")
	verbose   = flag.Bool("v", false, "verbose will print extra information about the build process")
	race      = flag.Bool("race", false, "build with -race, warning: enables CGO")
	pprof     = flag.Bool("pprof", false, "build with pprof support")
	uroot     = flag.String("uroot", "", "generate a source tree of kraken that can be embedded into u-root; requires dst path")
)

// config
var cfg *Config

// Target spec
type Target struct {
	Os   string
	Arch string
}

// Config yaml file structure
type Config struct {
	Pprof      bool
	Targets    map[string]Target
	Extensions []string
	Modules    []string
	Instances  []struct {
		Name      string
		Module    string
		Config    interface{}
		Requires  map[string]string
		Excludes  map[string]string
		Mutations map[string]struct {
			Mutates map[string]struct {
				From string
				To   string
			}
			Requires map[string]string
			Excludes map[string]string
		}
	}
}

// Simple search and replace
// Search all lines in a file and replace all instances of srchStr
// with replStr. ONLY WORKS FOR REGULAR FILES! Use DeepSearchandReplace()
// if directories must be considered.
func SimpleSearchAndReplace(filename, srchStr, replStr string) (e error) {
	// Try to open file
	input, e := ioutil.ReadFile(filename)
	if e != nil {
		log.Fatalf("error opening file %s for search and replace", filename)
	}

	// Put lines of file into array
	lines := strings.Split(string(input), "\n")

	// Iterate through lines to search and replace
	for i, line := range lines {
		if strings.Contains(line, srchStr) {
			// Replace srchStr with replStr in line[i]
			lines[i] = strings.ReplaceAll(string(lines[i]), srchStr, replStr)
		}
	}

	// Merge array into one string
	output := strings.Join(lines, "\n")

	// Attempt to write new contents to file
	e = ioutil.WriteFile(filename, []byte(output), 0644)
	if e != nil {
		log.Fatal("error writing file during search and replace")
	}
	return
}

// Deep search and replace
// If filename is a directory, traverse it recursively until regular
// files are reached, then perform a search and replace on them until
// all files under the directory are searched and replaced.
func DeepSearchAndReplace(filename, srchStr, replStr string) (e error) {
	// Get info of filename
	var info os.FileInfo
	info, e = os.Lstat(filename)
	if e != nil {
		return
	}

	// Is this a directory?
	if info.IsDir() {
		// If so, read each child and recurse until regular file found.
		var contents []os.FileInfo
		contents, e = ioutil.ReadDir(filename)
		for _, content := range contents {
			e = DeepSearchAndReplace(filepath.Join(filename, content.Name()), srchStr, replStr)
			if e != nil {
				return
			}
		}
	} else {
		// If not, perform a simple search and replace on the file
		e = SimpleSearchAndReplace(filename, srchStr, replStr)
	}

	return
}

func compileTemplate(tplFile, tmpDir string) (target string, e error) {
	var tpl *template.Template
	var out *os.File
	parts := strings.Split(filepath.Base(tplFile), ".")
	target = strings.Join(parts[:len(parts)-1], ".")
	if tpl, e = template.ParseFiles(tplFile); e != nil {
		return
	}
	if out, e = os.Create(filepath.Join(tmpDir, target)); e != nil {
		return
	}
	defer out.Close()
	e = tpl.Execute(out, cfg)
	return
}

func compileTemplates(krakenDir, tmpDir string) (targets []string, e error) {
	var files []os.FileInfo
	re, _ := regexp.Compile(".*\\.go\\.tpl$")
	// build a list of all of the templates
	files, e = ioutil.ReadDir(filepath.Join(krakenDir, "kraken"))
	if e != nil {
		return
	}
	for _, f := range files {
		if f.Mode().IsRegular() {
			if re.MatchString(f.Name()) { // ends in .go.tpl?
				if *verbose {
					log.Printf("executing template: %s", f.Name())
				}
				var target string
				target, e = compileTemplate(filepath.Join(krakenDir, "kraken", f.Name()), tmpDir)
				if e != nil {
					return
				}
				targets = append(targets, target)
			}
		}
	}
	return
}

func uCompileTemplates(krakenDir, tmpDir string) (targets []string, e error) {
	var files []os.FileInfo
	re, _ := regexp.Compile(".*\\.go\\.tpl$")
	// build a list of all of the templates
	files, e = ioutil.ReadDir(filepath.Join(krakenDir, "ukraken"))
	if e != nil {
		return
	}
	for _, f := range files {
		if f.Mode().IsRegular() {
			if re.MatchString(f.Name()) { // ends in .go.tpl?
				if *verbose {
					log.Printf("executing template: %s", f.Name())
				}
				var target string
				target, e = compileTemplate(filepath.Join(krakenDir, "ukraken", f.Name()), tmpDir)
				if e != nil {
					return
				}

				// Avoid import path errors in generated source files
				e = DeepSearchAndReplace(filepath.Join(tmpDir, target), "hpc/kraken", "u-root/u-root/cmds/exp/kraken")
				if e != nil {
					return
				}

				targets = append(targets, target)
			}
		}
	}
	return
}

// Generate kraken source tree for u-root command from krakenDir into outDir
func uKraken(outDir string, krakenDir string) (targets []string, e error) {
	// Create output directory if nonexistent
	e = os.MkdirAll(outDir, 0755)
	if *verbose {
		if e != nil {
			log.Printf("error locating/creating directory for u-root-embeddable kraken source tree")
			return
		} else {
			log.Printf("created/found directory \"%s\" for generated kraken source tree", outDir)
		}
	}

	// Create src dir (within outDir) for template compilation output
	srcDir := filepath.Join(outDir, "src") // make an option to change where this is?
	os.Mkdir(srcDir, 0755)

	// Generate kraken source from templates into outDir
	_, e = uCompileTemplates(krakenDir, srcDir)
	if *verbose {
		if e != nil {
			log.Printf("error compiling templates for u-root-embeddable kraken source tree")
			return
		} else {
			log.Printf("generated kraken source tree for u-root in %s", outDir)
		}
	}

	// Copy needed files from krakenDir to outDir
	files := []string{"config", "core", "extensions", "kraken", "lib", "modules", "utils", "vendor", "go.mod", "go.sum"}
	for _, file := range files {
		if *verbose {
			log.Printf("copying \"%s\" to \"%s\"", file, outDir)
		}
		inFile := path.Join(krakenDir, file)
		outFile := path.Join(outDir, file)
		e = cp.Copy(inFile, outFile)
		if e != nil {
			return
		}

		// Avoid import errors in generated source by modifying include path
		e = DeepSearchAndReplace(outFile, "hpc/kraken", "u-root/u-root/cmds/exp/kraken")
		if e != nil {
			return
		}
	}

	return
}

func buildKraken(dir string, fromTemplates []string, t Target, verbose bool) (e error) {
	// setup log file
	var f *os.File
	if verbose {
		f = os.Stderr
	} else {
		f, e = os.OpenFile(filepath.Join(dir, "log"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		defer f.Close()
	}

	args := []string{"build", "-o", "main"}
	if *race {
		args = append(args, "-race")
	}
	args = append(args, fromTemplates...)
	cmd := exec.Command("go", args...)
	if verbose {
		log.Printf("Run: %s", strings.Join(cmd.Args, " "))
	}
	cmd.Dir = dir

	cmd.Env = os.Environ()
	if *race {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
	} else {
		cmd.Env = append(cmd.Env, "CGO_ENABLED=0")
	}
	cmd.Env = append(cmd.Env, "GOOS="+t.Os)
	cmd.Env = append(cmd.Env, "GOARCH="+t.Arch)
	cmd.Env = append(cmd.Env, "GOPATH="+build.Default.GOPATH)
	cmd.Env = append(cmd.Env, "GOROOT="+build.Default.GOROOT)

	cmd.Stdout = f
	cmd.Stderr = f
	e = cmd.Run()
	return
}

func getModDir() (d string, e error) {
	// first, are we sitting in the Krakend dir?
	pwd, _ := os.Getwd()
	var f *os.File
	if f, e = os.Open(path.Join(pwd, "go.mod")); e == nil {
		defer f.Close()
		rd := bufio.NewReader(f)
		var line []byte
		if line, _, e = rd.ReadLine(); e == nil {
			if string(line) == KrModStr {
				d = pwd
				return
			}
		}
	}

	// couldn't open go.mod; obviously not in pwd, try for GOPATh
	var p *build.Package
	if p, e = build.Default.Import("github.com/hpc/kraken", "", build.FindOnly); e == nil {
		d = p.Dir
		return
	}
	e = fmt.Errorf("couldn't find craken in either PWD or GOPATH")
	return
}

func main() {
	var e error
	flag.Parse()

	// read config
	cfgBytes, e := ioutil.ReadFile(*cfgFile)
	if e != nil {
		log.Fatalf("could not read config file: %v", e)
	}
	cfg = &Config{}
	if e = yaml.Unmarshal(cfgBytes, cfg); e != nil {
		log.Fatalf("could not read config: %v", e)
	}
	if *pprof {
		cfg.Pprof = true
	}

	// Get kraken source tree root (and module directory)
	krakenDir, e := getModDir()
	if e != nil {
		log.Fatalf("error getting current module directory: %v", e)
	}
	log.Printf("using kraken at: %s", krakenDir)

	// Do we want to build source for u-root?
	var ufromTemplates []string
	if *uroot != "" {
		log.Printf("generating kraken source tree for u-root into \"%s\"", *uroot)
		ufromTemplates, e = uKraken(*uroot, krakenDir)
		if e != nil {
			log.Fatalf("could not create source tree for u-root: %v", e)
		}
	}

	// create build dir
	if _, e = os.Stat(*buildDir); os.IsNotExist(e) {
		if e = os.Mkdir(*buildDir, 0755); e != nil {
			log.Fatalf("could not create build directory: %v", e)
		}
	}

	tmpDir := filepath.Join(krakenDir, "tmp") // make an option to change where this is?
	os.Mkdir(tmpDir, 0755)

	// Create u-root kraken build dir if flag passed
	var ubldDir string // Build dir for kraken u-root command
	var usrcDir string // Source dir for kraken u-root compiled templates
	if *uroot != "" {
		ubldDir = filepath.Join(*uroot, "build")
		usrcDir = filepath.Join(*uroot, "src")

		// Make sure usrcDir exists
		if _, e = os.Stat(usrcDir); os.IsNotExist(e) {
			if e = os.Mkdir(usrcDir, 0755); e != nil {
				os.Mkdir(usrcDir, 0755)
			}
		}

		// Make sure ubldDir exists
		if _, e = os.Stat(ubldDir); os.IsNotExist(e) {
			if e = os.Mkdir(ubldDir, 0755); e != nil {
				log.Fatalf("could not create u-root kraken build directory: %v", e)
			}
		}
	}

	// setup build environment
	log.Println("setting up build environment")

	// build templates
	var fromTemplates []string
	if fromTemplates, e = compileTemplates(krakenDir, tmpDir); e != nil {
		log.Fatalf("could not compile templates: %v", e)
	}

	// build
	for t := range cfg.Targets {
		log.Printf("building: %s (GOOS: %s, GOARCH; %s)", t, cfg.Targets[t].Os, cfg.Targets[t].Arch)

		// Kraken for u-root
		if *uroot != "" {
			// Build
			log.Printf("building kraken for u-root")
			if e = buildKraken(usrcDir, ufromTemplates, cfg.Targets[t], *verbose); e != nil {
				log.Printf("failed to build %s for u-root: %v", t, e)
				continue
			}

			// Move binary to proper location
			upathMain := filepath.Join(usrcDir, "main")
			upathBuildDir := filepath.Join(*uroot, "build")
			upathKraken := filepath.Join(upathBuildDir, "kraken-"+t)
			if e = os.Rename(upathMain, upathKraken); e != nil {
				log.Printf("rename failed: %v", e)
			}
		}

		// Kraken proper
		if e = buildKraken(tmpDir, fromTemplates, cfg.Targets[t], *verbose); e != nil {
			log.Printf("failed to build %s: %v", t, e)
			continue
		}
		path := filepath.Join(*buildDir, "kraken-"+t)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			if *force {
				log.Printf("force was specified, overwriting old build: %s", path)
				os.Remove(path)
			} else {
				log.Printf("refusing to overwrite old build, use -force to override: %s", path)
			}
		}
		if e = os.Link(filepath.Join(tmpDir, "main"), path); e != nil {
			log.Printf("failed to link executable %s: %v", path, e)
			continue
		}
	}

	if !*noCleanup { // cleanup now
		os.RemoveAll(tmpDir)
		if *uroot != "" {
			os.RemoveAll(usrcDir)
		}
	} else {
		log.Printf("leaving temp directory: %s", tmpDir)
	}
}
