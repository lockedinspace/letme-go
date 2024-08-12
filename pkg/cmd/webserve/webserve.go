package webserve

import (
	"fmt"
	utils "github.com/lockedinspace/letme/pkg"
	letme "github.com/lockedinspace/letme/pkg/cmd"
	"github.com/spf13/cobra"
	"net/http"
	"os/exec"
)

var WebserveCmd = &cobra.Command{
	Use:   "webserve",
	Short: "Use letme with a graphic environment.",
	Long:  `Spin up a webserver which will enable the user to interact with letme graphically.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Starting server at port 8080\n")
		fileServer := http.FileServer(http.Dir("./pkg/cmd/webserve/static")) 
    	http.Handle("/", fileServer) 
		http.HandleFunc("/version", versionHandler)
		http.HandleFunc("/contexts", contextHandler)

		if err := http.ListenAndServe(":8080", nil); err != nil {
			utils.CheckAndReturnError(err)
		}
		
	},
}
func versionHandler(w http.ResponseWriter, r *http.Request) {
	// Execute the command
	output, err := exec.Command("letme", "--version").CombinedOutput()
	utils.CheckAndReturnError(err)
	// Send the output as JSON
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func contextHandler(w http.ResponseWriter, r *http.Request) {
	// Execute the command
	output, err := exec.Command("letme", "config", "get-contexts", "-o", "json").CombinedOutput()
	utils.CheckAndReturnError(err)
	// Send the output as JSON
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, `%s`, string(output))
}
func pingHandler(w http.ResponseWriter, r *http.Request) {
    if r.URL.Path != "/ping" {
        http.Error(w, "404 not found.", http.StatusNotFound)
        return
    }

    if r.Method != "GET" {
        http.Error(w, "Method is not supported.", http.StatusNotFound)
        return
    }

	fmt.Fprintf(w, "Pong!" )
}

func init() {
	letme.RootCmd.AddCommand(WebserveCmd)
}
