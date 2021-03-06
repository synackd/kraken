/* APIServer.go: provides the RPC API.  All gRPC calls live here
 * (except PhoneHome, which is a special exception in StateSyncEngine.go)
 *
 * Author: J. Lowell Wofford <lowell@lanl.gov>
 *
 * This software is open source software available under the BSD-3 license.
 * Copyright (c) 2018, Triad National Security, LLC
 * See LICENSE file for details.
 */

//go:generate protoc -I proto -I proto/include --go_out=plugins=grpc:proto proto/API.proto

package core

import (
	"context"
	"fmt"
	"net"

	"github.com/golang/protobuf/ptypes"

	"github.com/golang/protobuf/ptypes/empty"
	pb "github.com/hpc/kraken/core/proto"
	"github.com/hpc/kraken/lib"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

///////////////////////
// Auxiliary objects /
/////////////////////

// DiscoveryEvents announce a discovery
// This should probably live elsewhere
// This maps directly to a pb.DiscoveryControl
type DiscoveryEvent struct {
	ID      string // ID of a service instance
	URL     string // fully qualified, with node
	ValueID string
}

func (de *DiscoveryEvent) String() string {
	return fmt.Sprintf("(%s) %s == %s", de.ID, de.URL, de.ValueID)
}

//////////////////////
// APIServer Object /
////////////////////

var _ pb.APIServer = (*APIServer)(nil)

// APIServer is the gateway for gRPC calls into Kraken (i.e. the Module interface)
type APIServer struct {
	nlist net.Listener
	ulist net.Listener
	query *QueryEngine
	log   lib.Logger
	em    lib.EventEmitter
	sm    lib.ServiceManager
	schan chan<- lib.EventListener
	self  lib.NodeID
}

// NewAPIServer creates a new, initialized API
func NewAPIServer(ctx Context) *APIServer {
	api := &APIServer{
		nlist: ctx.RPC.NetListner,
		ulist: ctx.RPC.UNIXListener,
		query: &ctx.Query,
		log:   &ctx.Logger,
		em:    NewEventEmitter(lib.Event_API),
		schan: ctx.SubChan,
		self:  ctx.Self,
		sm:    ctx.Sm,
	}
	api.log.SetModule("API")
	return api
}

func (s *APIServer) QueryCreate(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	pbin := in.GetNode()
	out = &pb.Query{}
	if pbin == nil {
		e = fmt.Errorf("create query must contain a valid node")
		return
	}
	nin := NewNodeFromMessage(pbin)
	var nout lib.Node
	nout, e = s.query.Create(nin)
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryRead(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var nout lib.Node
	out = &pb.Query{}
	nout, e = s.query.Read(NewNodeIDFromURL(in.URL))
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryReadDsc(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var nout lib.Node
	out = &pb.Query{}
	nout, e = s.query.ReadDsc(NewNodeIDFromURL(in.URL))
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryUpdate(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	pbin := in.GetNode()
	out = &pb.Query{}
	if pbin == nil {
		e = fmt.Errorf("update query must contain a valid node")
		return
	}
	nin := NewNodeFromMessage(pbin)
	var nout lib.Node
	nout, e = s.query.Update(nin)
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryUpdateDsc(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	pbin := in.GetNode()
	out = &pb.Query{}
	if pbin == nil {
		e = fmt.Errorf("update query must contain a valid node")
		return
	}
	nin := NewNodeFromMessage(pbin)
	var nout lib.Node
	nout, e = s.query.UpdateDsc(nin)
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryDelete(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var nout lib.Node
	out = &pb.Query{}
	nout, e = s.query.Delete(NewNodeIDFromURL(in.URL))
	out.URL = in.URL
	if nout != nil {
		out.Payload = &pb.Query_Node{Node: nout.Message().(*pb.Node)}
	}
	return
}

func (s *APIServer) QueryReadAll(ctx context.Context, in *empty.Empty) (out *pb.QueryMulti, e error) {
	var nout []lib.Node
	out = &pb.QueryMulti{}
	out.Queries = []*pb.Query{}
	nout, e = s.query.ReadAll()
	for _, n := range nout {
		q := &pb.Query{
			URL: n.ID().String(),
			Payload: &pb.Query_Node{
				Node: n.Message().(*pb.Node),
			},
		}
		out.Queries = append(out.Queries, q)
	}
	return
}

func (s *APIServer) QueryReadAllDsc(ctx context.Context, in *empty.Empty) (out *pb.QueryMulti, e error) {
	var nout []lib.Node
	out = &pb.QueryMulti{}
	out.Queries = []*pb.Query{}
	nout, e = s.query.ReadAllDsc()
	for _, n := range nout {
		q := &pb.Query{
			URL: n.ID().String(),
			Payload: &pb.Query_Node{
				Node: n.Message().(*pb.Node),
			},
		}
		out.Queries = append(out.Queries, q)
	}
	return
}

func (s *APIServer) QueryMutationNodes(ctx context.Context, in *empty.Empty) (out *pb.Query, e error) {
	var mnlout pb.MutationNodeList
	url := "/graph/nodes"
	out = &pb.Query{}
	mnlout, e = s.query.ReadMutationNodes(url)
	out.URL = url
	if mnlout.MutationNodeList != nil {
		out.Payload = &pb.Query_MutationNodeList{
			MutationNodeList: &mnlout,
		}
	}
	return
}

func (s *APIServer) QueryMutationEdges(ctx context.Context, in *empty.Empty) (out *pb.Query, e error) {
	var melout pb.MutationEdgeList
	url := "/graph/nodes"
	out = &pb.Query{}
	melout, e = s.query.ReadMutationEdges(url)
	out.URL = "/graph/nodes"
	if melout.MutationEdgeList != nil {
		out.Payload = &pb.Query_MutationEdgeList{
			MutationEdgeList: &melout,
		}
	}
	return
}

func (s *APIServer) QueryNodeMutationNodes(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var mnlout pb.MutationNodeList
	out = &pb.Query{}
	mnlout, e = s.query.ReadNodeMutationNodes(in.URL)
	out.URL = in.URL
	if mnlout.MutationNodeList != nil {
		out.Payload = &pb.Query_MutationNodeList{
			MutationNodeList: &mnlout,
		}
	}
	return
}

func (s *APIServer) QueryNodeMutationEdges(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var melout pb.MutationEdgeList
	out = &pb.Query{}
	melout, e = s.query.ReadNodeMutationEdges(in.URL)
	out.URL = in.URL
	if melout.MutationEdgeList != nil {
		out.Payload = &pb.Query_MutationEdgeList{
			MutationEdgeList: &melout,
		}
	}
	return
}

func (s *APIServer) QueryNodeMutationPath(ctx context.Context, in *pb.Query) (out *pb.Query, e error) {
	var mpout pb.MutationPath
	out = &pb.Query{}
	mpout, e = s.query.ReadNodeMutationPath(in.URL)
	out.URL = in.URL
	if mpout.Chain != nil {
		out.Payload = &pb.Query_MutationPath{
			MutationPath: &mpout,
		}
	}
	return
}

func (s *APIServer) QueryDeleteAll(ctx context.Context, in *empty.Empty) (out *pb.QueryMulti, e error) {
	var nout []lib.Node
	out = &pb.QueryMulti{}
	out.Queries = []*pb.Query{}
	nout, e = s.query.DeleteAll()
	for _, n := range nout {
		q := &pb.Query{
			URL: n.ID().String(),
			Payload: &pb.Query_Node{
				Node: n.Message().(*pb.Node),
			},
		}
		out.Queries = append(out.Queries, q)
	}
	return
}

func (s *APIServer) QueryFreeze(ctx context.Context, in *empty.Empty) (out *pb.Query, e error) {
	e = s.query.Freeze()
	out = &pb.Query{}
	return
}
func (s *APIServer) QueryThaw(ctx context.Context, in *empty.Empty) (out *pb.Query, e error) {
	e = s.query.Thaw()
	out = &pb.Query{}
	return
}
func (s *APIServer) QueryFrozen(ctx context.Context, in *empty.Empty) (out *pb.Query, e error) {
	out = &pb.Query{}
	rb, e := s.query.Frozen()
	out.Payload = &pb.Query_Bool{Bool: rb}
	return
}

/*
 * Service management
 */

func (s *APIServer) ServiceInit(sir *pb.ServiceInitRequest, stream pb.API_ServiceInitServer) (e error) {
	srv := s.sm.GetService(sir.GetId())

	self, _ := s.query.Read(s.self)
	any, _ := ptypes.MarshalAny(self.Message())
	stream.Send(&pb.ServiceControl{
		Command: pb.ServiceControl_INIT,
		Config:  any,
	})
	c := make(chan lib.ServiceControl)
	srv.SetCtl(c)
	for {
		ctl := <-c
		stream.Send(&pb.ServiceControl{
			Command: pb.ServiceControl_Command(ctl.Command),
		})
	}
}

/*
 * Mutation management
 */

// MutationInit handles establishing the mutation stream
// This just caputures (filtered) mutation events and sends them over the stream
func (s *APIServer) MutationInit(sir *pb.ServiceInitRequest, stream pb.API_MutationInitServer) (e error) {
	sid := sir.GetId()
	echan := make(chan lib.Event)
	list := NewEventListener("MutationFor:"+sid, lib.Event_STATE_MUTATION,
		func(e lib.Event) bool {
			d := e.Data().(*MutationEvent)
			if d.Mutation[0] == sid {
				return true
			}
			return false
		},
		func(v lib.Event) error { return ChanSender(v, echan) })
	// subscribe our listener
	s.schan <- list

	for {
		v := <-echan
		smev := v.Data().(*MutationEvent)
		mc := &pb.MutationControl{
			Module: smev.Mutation[0],
			Id:     smev.Mutation[1],
			Type:   smev.Type,
			Cfg:    smev.NodeCfg.Message().(*pb.Node),
			Dsc:    smev.NodeDsc.Message().(*pb.Node),
		}
		if e := stream.Send(mc); e != nil {
			s.Logf(INFO, "mutation stream closed: %v", e)
			break
		}
	}

	// politely unsubscribe
	list.SetState(lib.EventListener_UNSUBSCRIBE)
	s.schan <- list
	return
}

// EventInit handles establishing the event stream
// This just caputures all events and sends them over the stream
func (s *APIServer) EventInit(sir *pb.ServiceInitRequest, stream pb.API_EventInitServer) (e error) {
	module := sir.GetModule()
	echan := make(chan lib.Event)
	filterFunction := func(e lib.Event) bool {
		return true
	}
	list := NewEventListener("EventFor:"+module, lib.Event_ALL,
		filterFunction,
		func(v lib.Event) error { return ChanSender(v, echan) })
	// subscribe our listener
	s.schan <- list

	for {
		v := <-echan
		var ec = &pb.EventControl{}
		switch v.Type() {
		case lib.Event_STATE_MUTATION:
			smev := v.Data().(*MutationEvent)
			ec = &pb.EventControl{
				Type: pb.EventControl_Mutation,
				Event: &pb.EventControl_MutationControl{
					MutationControl: &pb.MutationControl{
						Module: smev.Mutation[0],
						Id:     smev.Mutation[1],
						Type:   smev.Type,
						Cfg:    smev.NodeCfg.Message().(*pb.Node),
						Dsc:    smev.NodeDsc.Message().(*pb.Node),
					},
				},
			}
		case lib.Event_STATE_CHANGE:
			scev := v.Data().(*StateChangeEvent)
			s.Logf(lib.LLDEBUG, "api server got state change event: %+v\n%v", scev, scev.Value)
			ec = &pb.EventControl{
				Type: pb.EventControl_StateChange,
				Event: &pb.EventControl_StateChangeControl{
					StateChangeControl: &pb.StateChangeControl{
						Type:  scev.Type,
						Url:   scev.URL,
						Value: lib.ValueToString(scev.Value),
					},
				},
			}
		case lib.Event_DISCOVERY:
			dev := v.Data().(*DiscoveryEvent)
			ec = &pb.EventControl{
				Type: pb.EventControl_Discovery,
				Event: &pb.EventControl_DiscoveryEvent{
					DiscoveryEvent: &pb.DiscoveryEvent{
						Id:      dev.ID,
						Url:     dev.URL,
						ValueId: dev.ValueID,
					},
				},
			}
		default:
			s.Logf(lib.LLERROR, "Couldn't convert Event into mutation, statechange, or discovery: %+v", v)
		}
		if e := stream.Send(ec); e != nil {
			s.Logf(INFO, "event stream closed: %v", e)
			break
		}
	}

	// politely unsubscribe
	list.SetState(lib.EventListener_UNSUBSCRIBE)
	s.schan <- list
	return
}

// DiscoveryInit handles discoveries from nodes
// This dispatches nodes
func (s *APIServer) DiscoveryInit(stream pb.API_DiscoveryInitServer) (e error) {
	for {
		dc, e := stream.Recv()
		if e != nil {
			s.Logf(INFO, "discovery stream closed: %v", e)
			break
		}
		dv := &DiscoveryEvent{
			ID:      dc.GetId(),
			URL:     dc.GetUrl(),
			ValueID: dc.GetValueId(),
		}
		v := NewEvent(
			lib.Event_DISCOVERY,
			dc.GetUrl(),
			dv)
		s.EmitOne(v)
	}
	return
}

// LoggerInit initializes and RPC logger stream
func (s *APIServer) LoggerInit(stream pb.API_LoggerInitServer) (e error) {
	for {
		msg, e := stream.Recv()
		if e != nil {
			s.Logf(INFO, "logger stream closted: %v", e)
			break
		}
		s.Logf(lib.LoggerLevel(msg.Level), "%s:%s", msg.Origin, msg.Msg)
	}
	return
}

// Run starts the API service listener
func (s *APIServer) Run(ready chan<- interface{}) {
	s.Log(INFO, "starting API")
	srv := grpc.NewServer()
	pb.RegisterAPIServer(srv, s)
	reflection.Register(srv)
	ready <- nil
	if e := srv.Serve(s.ulist); e != nil {
		s.Logf(CRITICAL, "couldn't start API service: %v", e)
		return
	}
}

////////////////////////////
// Passthrough Interfaces /
//////////////////////////

/*
 * Consume Logger
 */
var _ lib.Logger = (*APIServer)(nil)

func (s *APIServer) Log(level lib.LoggerLevel, m string) { s.log.Log(level, m) }
func (s *APIServer) Logf(level lib.LoggerLevel, fmt string, v ...interface{}) {
	s.log.Logf(level, fmt, v...)
}
func (s *APIServer) SetModule(name string)                { s.log.SetModule(name) }
func (s *APIServer) GetModule() string                    { return s.log.GetModule() }
func (s *APIServer) SetLoggerLevel(level lib.LoggerLevel) { s.log.SetLoggerLevel(level) }
func (s *APIServer) GetLoggerLevel() lib.LoggerLevel      { return s.log.GetLoggerLevel() }
func (s *APIServer) IsEnabledFor(level lib.LoggerLevel) bool {
	return s.log.IsEnabledFor(level)
}

/*
 * Consume an emitter, so we implement EventEmitter directly
 */
var _ lib.EventEmitter = (*APIServer)(nil)

func (s *APIServer) Subscribe(id string, c chan<- []lib.Event) error {
	return s.em.Subscribe(id, c)
}
func (s *APIServer) Unsubscribe(id string) error { return s.em.Unsubscribe(id) }
func (s *APIServer) Emit(v []lib.Event)          { s.em.Emit(v) }
func (s *APIServer) EmitOne(v lib.Event)         { s.em.EmitOne(v) }
func (s *APIServer) EventType() lib.EventType    { return s.em.EventType() }
