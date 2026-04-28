
# Principal <-> Agent Kubernetes Event Code Walkthrough

Note: 
- Documents like this tend to become out of date fairly quickly. This was written in August 2024.
- Complete accuracy is not guaranteed: I'm new to this codebase, myself.

## Agent -> Principal: Agent receives K8s events and transmits them to principal via GRPC

The agent runs on the workload (spoke) cluster, and communicates to the principal on the control plane (hub) cluster. The agent watches an Argo CD Namespace containing Argo CD Applications. Events (create/update/delete) from those Argo CD Applications are communicated to principal. 

### A) Inside the agent process: Agent sees Application CR change and communicates it to principal via GRPC/Protobuf

In these steps, we go from receiving the Kubernetes event from client-go, to send it to principal via GRPC/protobuf.

1) Kubernetes events are received via 'appInformer' (field) of AppInformer (struct) in 'internal/informer/application/appinformer.go'
    - Changes to Application CRs in the Argo CD namespace on the workload cluster will be received/detected by 'appInformer' within AppInformer.
        - appInformer is a SharedIndexInformer, which is a standard go-client interface/concept
    - The Add/Update/Delete events received here are then passed to callback functions specified to AppInformer, defined in NewAppInformer.
	- For example, one such callback function is 'addAppCreationToQueue'


2) Agent receives events from AppInformer/appInformer via callback methods in agent.go
	- The Agent struct is created by 'NewAgent', and there we've specified a set of callback functions that will be called when particular events occur.
	- These Agent callback functions are called based on which event is received:
	    - addAppCreationToQueue: called for Argo CD Application create events
	    - addAppUpdateToQueue: called for Argo CD Application update events
	    - addAppDeletionToQueue: delete events
	    - listAppCallback: generic list events
    
3) The callback functions (listed above) are defined in 'agent/outbound.go', and they each output the create/update/delete event into SendRecvQueues queue, specifically the 'outbox' queue. Events in this queue are transmitted to the principal via GRPC.

4) 'sender' in 'agent/connection.go' (of Agent struct) reads outbound events from the 'outbox' queue, converts them to protobuf message format, then sends them on GRPC stream to principal
	- From this point in the agent, the remaining code is largely just some generated GRPC/Protobuf machinery that is responsible for actually transmitting the data.
	- Continue reading from inside the principal process, below. 


### B) Inside the principal process: Principal receives event and applies the content to Argo CD Application CR

NOTE: For the purpose of simplicity, I am ignoring the auth part of the workflow here. TL;DR: Agent first authenticates with principal before agent events will be processed, and this auth is handled via auth-specific GRPC messages.

In these steps, we go from receiving an event message from an agent, to updating Argo CD Application CR on control plane, based on the content of that event.

5) Event data from agent above is (ultimately) received in 'recvFunc' (of Server), in 'principal/apis/eventstream/eventstream.go'
	- 'recvFunc' reads a single event, converts it from protobuf into desired format, and stores it in principal's 'inbox' queue for processing.

6) 'eventProcessor' function (of Server) in principal/event.go, is responsible for processing the agent event received from the 'inbox' queue
	- Events are read by 'eventProcessor', then processed by 'processRecvQueue', also in 'principal/event.go'.
	- As of this writing, only Application events are processed (AppProject support in the future). Application events are handled by...


7) 'processApplicationEvent' (of Server) in principal/event.go, is responsible for processing Application events. Behaviour differs based on the type of event, and the mode of the sending agent.
	- Ultimately, these functions call the ApplicationManager to update the status of the Argo CD Application CR.

8) 'Create/UpdateAutonomousApp/UpdateStatus/Delete' in ApplicationManager, under 'internal/manager/application/application.go' will determine how to update the Argo CD Application CR.
	- The functions decide what specific fields of the Application CR to update, depending on the event context.
	- There is logic within the functions to update the application, depending on whether or not the backend supports patching (as of this writing, there is only 1 backend and it DOES support patching)
    - AFAICT the patching and non-patching cases should ultimately do the same thing (it's just a matter of HOW)
	- AppManager decides what needs to change in the Application CR, then calls the Application backend to do it.

9) KubernetesBackend (struct) in 'internal/backend/kubernetes/application/kubernetes.go' is responsible for updating Application CRs in a Kubernetes Namespace
	- KubernetesBackend implements the 'internal/backend/interface.go' of Application. As of this writing, the Kubernetes backend is the only implementation. 
	- The KubernetesBackend backend (and the Application interface it implements) are responsible for writing .spec/.status/.operation changes to Argo CD Application CR
	- The actual create/write/delete operations themselves are performed by the standard client-go Application client 


---

## Principal -> Agent: Principal receives K8s events, and passes them to agent

In these steps, the principal will detect a change to an Application CR in a control plane namespace, and transmit that change to the corresponding connect agent.

### A) In the principal process: principal detects event and transmits it to agent

1) AppInformer (struct) in 'internal/informer/application/appinformer.go' receives Kubernetes event (via appInformer)
	- Changes to Application CRs in the Argo CD namespace on the control plane cluster (hub cluster) will be received by 'appInformer' within AppInformer.
	- The 'Add/Update/Delete' events received here are then passed to functions specified to AppInformer, defined in 'NewAppInformer'.

2) Principal receives events from AppInformer via callback methods that are set in 'principal/server.go'
	- In 'NewServer' in server.go, the principal has specified a set of callback functions that will be called when particular events occur.
	- These Server functions are called based on which event is received:
	    - newAppCallback: Called when 'Create' events are received
	    - updateAppCallback: Called when 'Update' events are received
	    - deleteAppCallback: Called when 'Delete' events are received

3) The callback functions (listed above) defined in 'principal/callbacks.go' will output the specific event into SendRecvQueues queue, specifically the 'outbox' queue (SendQ). Events in this queue are transmitted to the agent via GRPC.

4) 'sendFunc' in 'principal/api/eventstream/eventstream.go' of Server reads outbound events from the 'outbox' queue (SendQ), converts them to protobuf message format, then sends them on GRPC stream to principal
	- From this point in the principal, the remaining code is largely generated GRPC/Protobuf machinery that actually transmits the data.
	- Continue reading from inside the agent process, below. 

### B) In the Agent process: agent receives event from principal, and processes it

In these steps, the agent will receive principal event, process it, and update the corresponding Argo CD Application cR.

5) 'receiver' (of Agent struct) in 'agent/connection.go' reads the GRPC event from the wire, and passes it to 'processIncomingEvent'

6) 'processIncomingEvent' (of Agent) in 'agent/inbound.go' calls 'processIncomingApplication', which calls 'createApplication/updateApplication/deleteApplication' (depending on the event type)

7) 'createApplication/updateApplication/deleteApplication' (of Agent) in 'agent/inbound.go' calls ApplicationManager
	- These functions will first sanity check the event, then call the corresponding ApplicationManager function to create/update/delete the Argo CD Application CR
	- The particular mode that the Agent is running in is relevant to which ApplicationManager function is called.
	    -  For example, for Update events:
	        - if the Agent is in Managed Mode, called UpdateManagedApp of ApplicationManager
	        - if the Agent is in Autonomous Mode, call UpdateOperation of ApplicationManager

Note: These next two steps are largely the same as the agent -> principal case.

8) Create/UpdateManagedApp/UpdateOperation/Delete in ApplicationManager, under 'internal/manager/application/application.go', determine how to update Argo CD Application cR
	- These functions within ApplicationManager will determine how to update the Argo CD Application CR.
	- The functions decide what specific fields of the Application CR to update, depending on the event context.
	- There is logic within the functions to update the Application CR, depending on whether or not the backend supports patching (as of this writing, there is only 1 backend and it DOES support patching)
	    - AFAICT the patching and non-patching cases should ultimately do the same thing (it's just a matter of method)
	- AppManager decides what needs to change in the Application CR, then calls the Application backend to do it.

9) KubernetesBackend in 'internal/backend/kubernetes/application/kubernetes.go' is responsible for updating Application CRs in agent's Argo CD Namespace
	- KubernetesBackend implements the 'internal/backend/interface.go' Application. As of this writing, the Kubernetes backend is the only implementation. 
	- The Application backend is responsible for writing changes to Argo CD Application CR in the Argo CD namespace that the agent is monitoring.
	- The actual create/write/delete operations themselves are performed by the standard client-go Application client 
