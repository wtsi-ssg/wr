package port

import (
	"fmt"
	"net"
)

const maxPort = 65535
const maxTries = maxPort - 10000

// Checker is used to check for available ports on a host.
type Checker struct {
	addr      *net.TCPAddr
	listeners []*net.TCPListener
	ports     map[int]bool
}

// NewChecker returns a Checker that can check ports on the given host.
func NewChecker(host string) (*Checker, error) {
	addr, err := net.ResolveTCPAddr("tcp", host+":0")
	if err != nil {
		return nil, err
	}

	return &Checker{
		addr:  addr,
		ports: make(map[int]bool),
	}, nil
}

// AvailableRange asks the operating system for ports until it is given a
// contiguous range of ports size long. It returns the first and last of those
// port numbers, having released all the ports it took, so they are ready for
// you to use.
//
// NB: there is the potential for a race condition here, where once released,
// another process gets one of the ports before you use it, so start listening
// on all the returned ports as soon as possible after calling this.
func (c *Checker) AvailableRange(size int) (int, int, error) {
	var err error
	defer func() {
		errr := c.release()
		if errr != nil {
			if err == nil {
				err = errr
			} else {
				err = fmt.Errorf("%w; %s", err, errr.Error())
			}
		}
	}()

	for i := 0; i < maxTries; i++ {
		port, erra := c.availablePort()
		if erra != nil {
			err = fmt.Errorf("%w; %s", err, erra.Error())
			continue
		}

		if set, has := c.checkRange(port, size); has {
			min, max := firstAndLast(set)
			return min, max, err
		}
	}

	return 0, 0, err
}

func (c *Checker) availablePort() (int, error) {
	l, err := net.ListenTCP("tcp", c.addr)
	if err != nil {
		return 0, err
	}
	c.listeners = append(c.listeners, l)
	port := l.Addr().(*net.TCPAddr).Port
	c.ports[port] = true

	return port, nil
}

func (c *Checker) checkRange(start, size int) ([]int, bool) {
	if len(c.ports) < size {
		return nil, false
	}

	after := c.portsAfter(start)
	if len(after)+1 >= size {
		return append([]int{start}, after[0:size-1]...), true
	}

	before := c.portsBefore(start)
	if len(before)+1 >= size {
		return append(before[len(before)-size+1:], start), true
	}

	if len(before)+len(after)+1 >= size {
		combined := append(before, append([]int{start}, after...)...)
		return combined[0:size], true
	}

	return nil, false
}

func (c *Checker) portsAfter(start int) []int {
	var ports []int
	for i := start + 1; i <= maxPort; i++ {
		if c.ports[i] {
			ports = append(ports, i)
		} else {
			return ports
		}
	}
	return ports
}

func (c *Checker) portsBefore(start int) []int {
	var ports []int
	for i := start - 1; i >= 1; i-- {
		if c.ports[i] {
			ports = append([]int{i}, ports...)
		} else {
			return ports
		}
	}
	return ports
}

func (c *Checker) release() error {
	var err error

	for _, l := range c.listeners {
		errl := l.Close()
		if errl != nil {
			err = fmt.Errorf("%w; %s", err, errl.Error())
		}
	}

	c.listeners = nil
	c.ports = make(map[int]bool)

	return err
}

func firstAndLast(s []int) (min, max int) {
	return s[0], s[len(s)-1]
}
