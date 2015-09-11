package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/chrissnell/lbaas/config"
	"github.com/chrissnell/lbaas/controller"
	"github.com/chrissnell/lbaas/model"
	"github.com/chrissnell/lbaas/model/loadbalancers"

	"github.com/coreos/etcd/client"
	"github.com/emicklei/go-restful"
)

// Service contains our configuration and runtime objects
type Service struct {
	Config     config.Config
	etcdclient client.Client
	controller *controller.Controller
	model      *model.Model
}

// New creates a new instance of Service with the given configuration file
func New(filename string) *Service {
	s := &Service{}

	// Read our server configuration
	filename, _ = filepath.Abs(filename)
	cfg, err := config.New(filename)
	if err != nil {
		log.Fatalln("Error reading config file.  Did you pass the -config flag?  Run with -h for help.\n", err)
	}
	s.Config = cfg

	// Open an etcd client
	var endpoints []string
	endpoints = append(endpoints, fmt.Sprintf("http://%v:%v", s.Config.Etcd.Hostname, s.Config.Etcd.Port))

	ec := client.Config{
		Endpoints: endpoints,
		Transport: client.DefaultTransport,
	}

	s.etcdclient, err = client.New(ec)
	if err != nil {
		log.Fatalln("Could not connect to etcd:", err)
	}

	// Initialize our model with load balancer based on our configured type
	switch s.Config.LoadBalancer.Kind {
	case "f5":
		s.model = model.New(s.etcdclient, f5.New(), s.Config)
	default:
		s.model = model.New(s.etcdclient, f5.New(), s.Config)
	}

	// Initialize the Controller
	s.controller = controller.New(s.Config, s.model)

	return s
}

// Listen will start the HTTP listeners for the API router.
func (s *Service) Listen() {

	restful.Add(s.controller.WS)

	log.Println("Listening on ", s.Config.Service.APIListenAddr)
	// Set up our API endpoint router
	log.Fatal(http.ListenAndServe(s.Config.Service.APIListenAddr, nil))
}

// Close will shut down the service
func (s *Service) Close() {
	// s.db.Close()
}

func main() {
	cfgFile := flag.String("config", "config.yaml", "Path to config file (default: ./config.yaml)")
	flag.Parse()

	s := New(*cfgFile)
	defer s.Close()
	s.Listen()
}
