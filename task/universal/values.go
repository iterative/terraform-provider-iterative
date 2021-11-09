package universal

import (
	"errors"
	"net"
	"time"
)

var NotFoundError error = errors.New("resource not found")
var NotImplementedError error = errors.New("resource method not implemented")

type Size struct {
	Storage int
	Machine string
}

type Status struct {
	Address net.IP
	Active  bool
}

type Event struct {
	Time        time.Time
	Code        string
	Description []string
}
type Storage struct{}

type Task struct {
	Size
	Environment
	Firewall
	Spot        float64 // <0=disabled, 0=auto, >0=fixed
	Parallelism uint16
	Tags        map[string]string // Deprecated

	Addresses []net.IP
	Status    map[string]int
	Events    []Event
}

// Firewall
type Firewall struct {
	Ingress FirewallRule
	Egress  FirewallRule
}

// Firewall rule: not specified fields mean "allow any"; sepcified but empty mean "allow none";
// ports are both TCP and UDP; when ports is not specified, it will allow ingress to every port
// and every protocol, not only TCP&UDP
type FirewallRule struct {
	Nets  *[]net.IPNet
	Ports *[]uint16
}
