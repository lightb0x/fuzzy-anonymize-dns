package main
import (
	"crypto/tls"
	"fmt"
	"time"
	"net/http"
	"math"
	"math/rand"
	"strings"
	"github.com/miekg/dns"
	"github.com/gin-gonic/gin"
	"github.com/patrickmn/go-cache"
)

func main() {
	// IP of fuzzy-anonymize-dns servers
	proxy_ip := []string{}
	rr_ip := "1.1.1.1:853"
	
	// number of expected (query_up) hops along chain
	const query_up_hop = 9
	query_up_prob := float32(
		2. +
		1. / query_up_hop -
		math.Sqrt(4. / query_up_hop + 1. / math.Pow(query_up_hop, 2))) / 2.
	fmt.Printf("p = %f\n", query_up_prob)

	// construct thread-safe cache
	// default 300 seconds to expire
	// purge every 10 min
	// invariant:
	// * cache contains only "validated" DN-IP pair from
	//   * recursive resolver AND/OR
	//   * other random proxy
	cs := cache.New(300 * time.Second, 10*time.Minute)

	r := gin.Default()

	// for testing
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	// DNS query
	// to be stateless while considering cache
	// if cache hit: return. else: conduct (query_up)
	r.GET("/query/:domain_name", func(c *gin.Context) {
		// get query domain name
		domain_name := c.Params.ByName("domain_name")
		if domain_name[len(domain_name) - 1] != '.' {
			domain_name = fmt.Sprintf("%s.", domain_name)
		}

		// read cache
		ip, found := cs.Get(domain_name)
		if found {
			c.JSON(http.StatusOK, gin.H{
				"IP": ip,
			})
			return
		}

		seed := rand.NewSource(time.Now().UnixNano())
		r1 := rand.New(seed).Float32()

		// query_up:
		// * query random proxy IP w/ probability p, AND
		if r1 < query_up_prob {
			seed := rand.NewSource(time.Now().UnixNano())
			r2 := rand.New(seed).Int()
			proxy := proxy_ip[r2 % len(proxy_ip)]
			
			proxy_detail := fmt.Sprintf("http://%s:8853/query/%s", proxy, domain_name)
			go http.Get(proxy_detail)
		}

		// * query recursive resolver
		//   * ? after random interval u in U[0, ~10] ms
		seed = rand.NewSource(time.Now().UnixNano())
		r3 := rand.New(seed).Float32()
		time.Sleep(time.Duration(int(r3 * 10)) * time.Millisecond)

		//   * query recursive resolver
		m := dns.Msg{}
		m.SetQuestion(domain_name, dns.TypeA)

		cl := dns.Client{}
		cl.Net = "tcp-tls"
		cl.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
		vr, _, err := cl.Exchange(&m, rr_ip)
		if err != nil {
			fmt.Printf("Error querying recursive resolver at %s with: %v\n", rr_ip, err)
			c.Status(http.StatusNotFound)
			return
		}
		if vr == nil {
			fmt.Printf("%s returned nil\n", rr_ip)
			c.Status(http.StatusNotFound)
			return
		}

		// write cache
		for i := 0; i < len(vr.Answer); i++ {
			item := vr.Answer[i]
			item_header := item.Header()
			item_dump := strings.Split(item.String(), "\t")
			item_data := item_dump[len(item_dump) - 1]

			name := item_header.Name
			ttl := item_header.Ttl

			// fmt.Println(name, ttl, item_data)

			cs.Set(name, item_data, time.Duration(ttl) * time.Second)
		}

		// verify response (skipped)

		// return response
		ip, found = cs.Get(domain_name)
		if found {
			c.JSON(http.StatusOK, gin.H{
				"IP": ip,
			})
			return
		}
	})
	r.Run("0.0.0.0:8853")
}
