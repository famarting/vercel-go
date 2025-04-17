package handler

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/dapr/go-sdk/client"
	"github.com/dapr/go-sdk/workflow"
)

func workflowHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		respondResult(w, "ok")
		return
	}
	ctx := r.Context()

	daprClient, err := client.NewClientWithAddressContext(ctx, os.Getenv("DAPR_GRPC_ENDPOINT"))
	if err != nil {
		respondError(w, err)
		return
	}
	defer daprClient.Close()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		respondError(w, err)
		return
	}
	fmt.Println("received request " + string(body))

	worker, err := workflow.NewWorker(workflow.WorkerWithDaprClient(daprClient))
	if err != nil {
		respondError(w, err)
		return
	}

	if err := worker.RegisterWorkflow(TestWorkflow); err != nil {
		respondError(w, err)
		return
	}
	if err := worker.RegisterActivity(TestActivity); err != nil {
		respondError(w, err)
		return
	}

	if err := worker.Start(); err != nil {
		respondError(w, err)
		return
	}
	fmt.Println("runner started")
	defer func() {
		// stop workflow runtime
		if err := worker.Shutdown(); err != nil {
			fmt.Println("failed to shutdown runtime: " + err.Error())
			return
		}
		fmt.Println("workflow worker successfully shutdown")
	}()

	wfClient, err := workflow.NewClient(workflow.WithDaprClient(daprClient))
	if err != nil {
		respondError(w, err)
		return
	}

	id, err := wfClient.ScheduleNewWorkflow(ctx, "TestWorkflow")
	if err != nil {
		respondError(w, err)
		return
	}
	fmt.Println("workflow started: " + id)

	res, err := wfClient.WaitForWorkflowCompletion(ctx, id)
	if err != nil {
		respondError(w, err)
		return
	}
	fmt.Println("workflow completed: " + res.SerializedOutput)
	respondJson(w, map[string]string{
		"id":     id,
		"status": res.RuntimeStatus.String(),
		"result": res.SerializedOutput,
	})
}

func TestWorkflow(ctx *workflow.WorkflowContext) (any, error) {
	var result string
	err := ctx.CallActivity(TestActivity).Await(result)
	if err != nil {
		return nil, err
	}
	return result, nil
}

func TestActivity(ctx workflow.ActivityContext) (any, error) {
	return "ok", nil
}
