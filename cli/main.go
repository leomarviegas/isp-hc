\
package main

// Minimal isp-checker CLI prototype (simulation mode).
// Usage:
//   go run main.go --simulation ./simulations/healthy.json
//
// It reads a simulation JSON and prints it to stdout (ensures schema fields exist).
import (
    "encoding/json"
    "flag"
    "fmt"
    "io/ioutil"
    "log"
    "os"
)

func main() {
    simPath := flag.String("simulation", "", "Path to simulation JSON file")
    outPath := flag.String("out", "", "Optional output path")
    flag.Parse()
    if *simPath == "" {
        log.Fatalf("please provide --simulation <file>")
    }
    b, err := ioutil.ReadFile(*simPath)
    if err != nil {
        log.Fatalf("read error: %v", err)
    }
    // Validate minimal schema by unmarshalling to map
    var m map[string]interface{}
    if err := json.Unmarshal(b, &m); err != nil {
        log.Fatalf("invalid json: %v", err)
    }
    // Add run_id if missing
    if _, ok := m["run_id"]; !ok {
        m["run_id"] = "generated-local"
    }
    out, _ := json.MarshalIndent(m, "", "  ")
    if *outPath != "" {
        if err := ioutil.WriteFile(*outPath, out, 0644); err != nil {
            log.Fatalf("write out error: %v", err)
        }
        fmt.Printf("Wrote result to %s\n", *outPath)
        return
    }
    os.Stdout.Write(out)
}
