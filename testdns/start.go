package main

import (
	"net"
	"strconv"
	"log"
	"github.com/dns"
	"fmt"
	"sync"
	"time"
)

var dnsAnswerMap sync.Map

var config, _ = dns.ClientConfigFromFile("/etc/resolv.conf")

type handler struct{}

func lookup(domain string, msg *dns.Msg, dnsType uint16) {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	key := domain + strconv.Itoa(int(dnsType))
	answer, _ := dnsAnswerMap.Load(key)

	if answer != nil {
		fmt.Println("Load answer from cache")
		msg.Answer = answer.([]dns.RR)
		return
	}

	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dnsType)
	m.RecursionDesired = true

	var c = new(dns.Client)
	c.ReadTimeout = time.Duration(5) * 1e9
	r, _, err := c.Exchange(m, net.JoinHostPort(config.Servers[0], config.Port))
	if r == nil {
		log.Fatalf("*** error: %s\n", err.Error())
	}
	if r.Rcode != dns.RcodeSuccess {
		log.Println(fmt.Printf(" *** invalid answer name %s after MX query for %s\n", domain, domain))
	}
	// Stuff must be in the answer section
	/*for _, a := range r.Answer {
		fmt.Printf("%v\n", a)
	}*/

	fmt.Println(key + " -> Store answer to cache")
	dnsAnswerMap.Store(key, r.Answer)
	msg.Answer = r.Answer
}

func (this *handler) ServeDNS(w dns.ResponseWriter, r *dns.Msg) {
	var do = make(chan bool)

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println(err)
			}
		}()
		msg := dns.Msg{}
		msg.SetReply(r)

		/*for _, e := range r.Question {
			fmt.Println(e.String())
		}*/

		//fmt.Println("dns.Type: " + strconv.Itoa(int(r.Question[0].Qtype)))

		domain := msg.Question[0].Name
		switch r.Question[0].Qtype {
		case dns.TypeA:
			msg.Authoritative = true
			lookup(domain, &msg, dns.TypeA)
		default:
			lookup(domain, &msg, r.Question[0].Qtype)
		}
		w.WriteMsg(&msg)
		do <- true
	}()

	<-do
}

func main() {
	defer func() {
		if err := recover(); err != nil {
			fmt.Println(err)
		}
	}()
	srv := &dns.Server{Addr: ":" + strconv.Itoa(53), Net: "udp"}
	srv.Handler = &handler{}
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Failed to set udp listener %s\n", err.Error())
	}
}
