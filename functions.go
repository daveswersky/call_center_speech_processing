package function

import (
	"fmt"
	"net/http"

	//"github.com/GoogleCloudPlatform/functions-framework-go/funcframework"
	//"github.com/GoogleCloudPlatform/functions-framework-go/functions"
)

func init() {
	//functions.HTTP("HelloWorld", helloWorld)
	//functions.CloudEvent("AudioProcess", process_transcript)
	//funcframework.RegisterEventFunctionContext()
}

// helloWorld writes "Hello, World!" to the HTTP response.
func helloWorld(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Hello, World!")
}
