package server

import (
	"protocol"
	"common"
	"net"
)

/////////////////////////////////////////////////////////////////////////////
// Type Declaration 
/////////////////////////////////////////////////////////////////////////////

type FollowerServer struct {
	follower     *protocol.Follower
	state        *FollowerState
}

type FollowerState struct {
	serverState			*ServerState
}

/////////////////////////////////////////////////////////////////////////////
// FollowerServer - Discovery Phase (synchronize with leader)
/////////////////////////////////////////////////////////////////////////////

//
// Create a new FollowerServer
//
func newFollowerServer(naddr string, 
					   leader string,
                       handler protocol.ActionHandler, 
                       factory protocol.MsgFactory) (*FollowerServer, error) {

	// create connection to leader
	conn, err := net.Dial("tcp", leader)
	if err != nil {
		return nil, err	
	}
	pipe, err := common.NewPeerPipe(conn)
	if err != nil {
		return nil, err
	}
	
	// create a follower 
	follower := protocol.NewFollower(protocol.FOLLOWER, pipe, handler, factory)
	
	// create the follower state
	state := newFollowerState() 
	                    
	// create the server					 
	server := &FollowerServer{follower : follower,
	                          state : state}
	
	// start sycrhorniziing with the leader
	go server.syncWithLeader(pipe, handler, factory)

	return server, nil
}

//
// Synchronize with the leader
//
func (s *FollowerServer) syncWithLeader(pipe      *common.PeerPipe,
										handler   protocol.ActionHandler, 
					                    factory   protocol.MsgFactory) {
	
	donech := make(chan bool)				
	proxy := protocol.NewFollowerSyncProxy(pipe, handler, factory)                    	 
	proxy.Start(donech)
	
	// This goroutine will block until NewFollowerSyncProxy has sychronized with
	// the leader (a bool is pushed to donech)
	<- donech

	//start processing request
	s.processRequest(handler, factory)
}

//
// Create a new FollowerState
//
func newFollowerState() *FollowerState {
	serverState := newServerState()
	serverState.setStatus(protocol.FOLLOWING)
	
	state := &FollowerState{serverState : serverState}
	return state                       
}

/////////////////////////////////////////////////////////////////////////////
// FollowerServer - Broadcast phase (handle request) 
/////////////////////////////////////////////////////////////////////////////

//
// Goroutine for processing each request one-by-one
//
func (s *FollowerServer) processRequest(handler   protocol.ActionHandler, 
					                    factory   protocol.MsgFactory) {
	for {
		// de-queue the request
		handle := <- s.state.serverState.incomings
		
		s.state.serverState.mutex.Lock()
		defer s.state.serverState.mutex.Unlock()
		
		// remember the request 		
		s.state.serverState.pendings[handle.request.GetReqId()] = handle
		
		// forward the request to the leader
		s.follower.ForwardRequest(handle.request)
	}
}

/////////////////////////////////////////////////////////////////////////////
// FollowerServer - StateProvider Interface 
/////////////////////////////////////////////////////////////////////////////

func (s *FollowerServer) GetState() *ServerState {
	return s.state.serverState
}