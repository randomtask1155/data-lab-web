package main

import (
  "net/http"
  "fmt"
  "os"
  "os/exec"
  "io/ioutil"
  "encoding/json"
  "strings"
  "net"
  "time"
  "sync"
  "html/template"
)

var (
  pingCommand = "ping"
  Rules = "resources/rules.json"
  RulesLock = sync.Mutex{}
  pingDeadline = "10"
  statusTimeout = 20
  CurrentIPStatus []IPStatus
  startPageTemplate = template.Must(template.ParseFiles("tmpl/start.tmpl")) // root page
)

type IPRules struct {
  IPRanges string `json:"ipranges"`
  PollingInterval int `json:"polling-interval"`
}

type ErrorResponse struct {
  ErrorMessage string `json:"ErrorMessage"`
}

type IPStatus struct {
  IP string `json:"ipaddr"`
  IsFree bool `json:"isfree"`
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
  startPageTemplate.Execute(w, "")
}

func getIPList(w http.ResponseWriter, r *http.Request) {
  w.Header().Set("Content-Type", "application/json")  
  status, err := json.Marshal(getCurrentStatus())
  if err != nil {
    w.Write(marshalErrorResponse(fmt.Sprintf("%s", err)))
    return
  }
  w.Write(status)
}


func monitorIPrange(interval int) {
  for {
    processIPrange()
    time.Sleep(time.Second * time.Duration(int64(interval)))
  }
}

func processIPrange() {
  
  ipr, err := loadRules()
  if err != nil {
    fmt.Printf("Failed to load rules: %s", err)
    return
  }
  ranges := strings.Split(ipr.IPRanges, ",")
  statusChan := make(chan IPStatus, 0)
  for i := range ranges {
    if strings.ContainsAny(ranges[i], "-") {
      startEnd := strings.Split(ranges[i], "-")
      if len(startEnd) != 2 {
        fmt.Sprintf("Invalid ip range detected: %s", ranges[i])
        return
      }
      startIP := net.ParseIP(startEnd[0])
      endIP := net.ParseIP(startEnd[1])
      maxLoop := 1000 // protect from infinite loop
      for {
          if maxLoop <= 0 {
            fmt.Println("Max Loop Reached!")
            break
          }
          if startIP.Equal(endIP) {
            go pingIP(startIP.String(), statusChan)
            break
          }
          go pingIP(startIP.String(), statusChan)
          startIP[len(startIP)-1] = startIP[len(startIP)-1] + 1
          maxLoop -= 1
      }
    } else {
      go pingIP(ranges[i], statusChan)
    }
  }
  
  IPStatuses := make([]IPStatus,0)
  for {
    select {
    case ip := <-statusChan:
      IPStatuses = append(IPStatuses, ip)
    case <-time.After(time.Second * time.Duration(int64(statusTimeout))):
      updateCurrentStatus(IPStatuses)
      return
    }
  }
    
}

func updateCurrentStatus(status []IPStatus) {
  RulesLock.Lock()
  CurrentIPStatus = status 
  RulesLock.Unlock()
}

func getCurrentStatus() []IPStatus {
  return CurrentIPStatus 
}

// return true if ip is up and false if ip in down
func pingIP(ip string, status chan IPStatus) {
  _, err := exec.Command(pingCommand, "-c", "5", "-w", pingDeadline, ip).CombinedOutput()
  if err != nil {
    status <- IPStatus{ip, false}
    return
  }
  status <- IPStatus{ip, true}
}

func marshalErrorResponse(s string) []byte {
  emsg := ErrorResponse{s}
  b,_ := json.Marshal(emsg)
  return b
}

func loadRules() (IPRules, error) {
  ipr := IPRules{}
  b, err := ioutil.ReadFile(Rules)
  if err != nil {
    return ipr, err
  }
  
  err = json.Unmarshal(b, &ipr)
  if err != nil {
    return ipr, err
  }
  return ipr, nil
}

func findCommands() {
  var err error
  pingCommand, err = exec.LookPath(pingCommand)
  if err != nil {
    fmt.Printf("ping lookup failed: %s\n", err)
    os.Exit(1)
  }
}

func main() {
  port := os.Getenv("PORT")
  if port == "" {
    port = "8080"
  }
  findCommands()
  
  rules, err := loadRules()
  if err != nil {
    fmt.Println(err)
    os.Exit(4)
  }
  go monitorIPrange(rules.PollingInterval)
  
  http.HandleFunc("/", rootHandler)
  http.HandleFunc("/api/get", getIPList)
  
  http.Handle("/img/", http.FileServer(http.Dir("")))
  http.Handle("/fonts/", http.FileServer(http.Dir("")))
  http.Handle("/js/", http.FileServer(http.Dir("")))
  http.Handle("/css/", http.FileServer(http.Dir("")))
  err = http.ListenAndServe(":"+port, nil)
  if err != nil {
    fmt.Printf("Failed to start http server: %s\n", err)
  }
}