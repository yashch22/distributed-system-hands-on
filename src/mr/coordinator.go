package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type TaskState int

const (
	TaskIdle TaskState = iota
	TaskInProgress
	TaskCompleted
)

type Task struct {
	State     TaskState
	StartTime time.Time
	FileNames []string // For map: single file, for reduce: all intermediate files
	MapTaskId int      // For reduce tasks
}

type Coordinator struct {
	// Your definitions here.
	mu          sync.Mutex
	mapTasks    []Task
	reduceTasks []Task
	nReduce     int
	nMap        int
	mapDone     bool
	allDone     bool
}

// Your code here -- RPC handlers for the worker to call.

func (c *Coordinator) GetTask(args *GetTaskArgs, reply *GetTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check for timeouts and reassign tasks
	c.checkTimeouts()

	// First, assign map tasks
	if !c.mapDone {
		for i, task := range c.mapTasks {
			if task.State == TaskIdle {
				c.mapTasks[i].State = TaskInProgress
				c.mapTasks[i].StartTime = time.Now()
				reply.TaskType = MapTask
				reply.TaskId = i
				reply.FileName = task.FileNames[0]
				reply.NReduce = c.nReduce
				return nil
			}
		}
		// Check if all maps are completed
		allMapsDone := true
		for _, task := range c.mapTasks {
			if task.State != TaskCompleted {
				allMapsDone = false
				break
			}
		}
		if allMapsDone {
			c.mapDone = true
		} else {
			// Maps are in progress, wait
			reply.TaskType = NoTask
			return nil
		}
	}

	// All maps done, assign reduce tasks
	for i, task := range c.reduceTasks {
		if task.State == TaskIdle {
			// Collect all intermediate files for this reduce task
			fileNames := []string{}
			for mapId := 0; mapId < c.nMap; mapId++ {
				fileName := fmt.Sprintf("mr-%d-%d", mapId, i)
				fileNames = append(fileNames, fileName)
			}
			c.reduceTasks[i].State = TaskInProgress
			c.reduceTasks[i].StartTime = time.Now()
			c.reduceTasks[i].FileNames = fileNames
			reply.TaskType = ReduceTask
			reply.TaskId = i
			reply.FileNames = fileNames
			reply.NReduce = c.nReduce
			return nil
		}
	}

	// Check if all reduces are done
	allReducesDone := true
	for _, task := range c.reduceTasks {
		if task.State != TaskCompleted {
			allReducesDone = false
			break
		}
	}

	if allReducesDone {
		c.allDone = true
		reply.TaskType = ExitTask
	} else {
		reply.TaskType = NoTask
	}

	return nil
}

func (c *Coordinator) ReportTask(args *ReportTaskArgs, reply *ReportTaskReply) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if args.TaskType == MapTask {
		if args.TaskId >= 0 && args.TaskId < len(c.mapTasks) {
			if args.Success {
				c.mapTasks[args.TaskId].State = TaskCompleted
			} else {
				// Task failed, reset to idle so it can be retried
				c.mapTasks[args.TaskId].State = TaskIdle
			}
		}
	} else if args.TaskType == ReduceTask {
		if args.TaskId >= 0 && args.TaskId < len(c.reduceTasks) {
			if args.Success {
				c.reduceTasks[args.TaskId].State = TaskCompleted
			} else {
				// Task failed, reset to idle so it can be retried
				c.reduceTasks[args.TaskId].State = TaskIdle
			}
		}
	}

	return nil
}

func (c *Coordinator) checkTimeouts() {
	timeout := 10 * time.Second
	now := time.Now()

	// Check map task timeouts
	for i := range c.mapTasks {
		if c.mapTasks[i].State == TaskInProgress {
			if now.Sub(c.mapTasks[i].StartTime) > timeout {
				c.mapTasks[i].State = TaskIdle
			}
		}
	}

	// Check reduce task timeouts
	for i := range c.reduceTasks {
		if c.reduceTasks[i].State == TaskInProgress {
			if now.Sub(c.reduceTasks[i].StartTime) > timeout {
				c.reduceTasks[i].State = TaskIdle
			}
		}
	}
}

// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	reply.Y = args.X + 1
	return nil
}

// start a thread that listens for RPCs from worker.go
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
func (c *Coordinator) Done() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.allDone
}

// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.
	c.nReduce = nReduce
	c.nMap = len(files)
	c.mapDone = false
	c.allDone = false

	// Initialize map tasks
	c.mapTasks = make([]Task, len(files))
	for i, file := range files {
		c.mapTasks[i] = Task{
			State:     TaskIdle,
			FileNames: []string{file},
		}
	}

	// Initialize reduce tasks
	c.reduceTasks = make([]Task, nReduce)
	for i := 0; i < nReduce; i++ {
		c.reduceTasks[i] = Task{
			State: TaskIdle,
		}
	}

	c.server()
	return &c
}
