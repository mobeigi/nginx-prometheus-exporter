package collector

import (
	"log"
	"sync"

	plusclient "github.com/nginxinc/nginx-plus-go-client/client"
	"github.com/prometheus/client_golang/prometheus"
)

// NginxPlusCollector collects NGINX Plus metrics. It implements prometheus.Collector interface.
type NginxPlusCollector struct {
	nginxClient                 *plusclient.NginxClient
	totalMetrics                map[string]*prometheus.Desc
	serverZoneMetrics           map[string]*prometheus.Desc
	upstreamMetrics             map[string]*prometheus.Desc
	upstreamServerMetrics       map[string]*prometheus.Desc
	streamServerZoneMetrics     map[string]*prometheus.Desc
	streamUpstreamMetrics       map[string]*prometheus.Desc
	streamUpstreamServerMetrics map[string]*prometheus.Desc
	streamZoneSyncMetrics       map[string]*prometheus.Desc
	locationZoneMetrics         map[string]*prometheus.Desc
	resolverMetrics             map[string]*prometheus.Desc
	upMetric                    prometheus.Gauge
	mutex                       sync.Mutex
	ICLabels                    ICLabels
}

// ICLabels stores Ingress Controller related label data.
type ICLabels struct {
	serverZoneLabels     map[string]string
	upstreamServerLabels map[string]string
}

// VariableLabels stores label names.
type VariableLabels struct {
	ServerZoneLabels     []string
	UpstreamServerLabels []string
}

// NewNginxPlusCollector creates an NginxPlusCollector.
func NewNginxPlusCollector(nginxClient *plusclient.NginxClient, namespace string, varLabels VariableLabels, constLabels map[string]string) *NginxPlusCollector {
	return &NginxPlusCollector{
		nginxClient: nginxClient,
		totalMetrics: map[string]*prometheus.Desc{
			"connections_accepted":  newGlobalMetric(namespace, "connections_accepted", "Accepted client connections", constLabels),
			"connections_dropped":   newGlobalMetric(namespace, "connections_dropped", "Dropped client connections", constLabels),
			"connections_active":    newGlobalMetric(namespace, "connections_active", "Active client connections", constLabels),
			"connections_idle":      newGlobalMetric(namespace, "connections_idle", "Idle client connections", constLabels),
			"http_requests_total":   newGlobalMetric(namespace, "http_requests_total", "Total http requests", constLabels),
			"http_requests_current": newGlobalMetric(namespace, "http_requests_current", "Current http requests", constLabels),
			"ssl_handshakes":        newGlobalMetric(namespace, "ssl_handshakes", "Successful SSL handshakes", constLabels),
			"ssl_handshakes_failed": newGlobalMetric(namespace, "ssl_handshakes_failed", "Failed SSL handshakes", constLabels),
			"ssl_session_reuses":    newGlobalMetric(namespace, "ssl_session_reuses", "Session reuses during SSL handshake", constLabels),
		},
		serverZoneMetrics: map[string]*prometheus.Desc{
			"processing":    newServerZoneMetric(namespace, "processing", "Client requests that are currently being processed", varLabels, constLabels),
			"requests":      newServerZoneMetric(namespace, "requests", "Total client requests", varLabels, constLabels),
			"responses_1xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx": newServerZoneMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"discarded":     newServerZoneMetric(namespace, "discarded", "Requests completed without sending a response", varLabels, constLabels),
			"received":      newServerZoneMetric(namespace, "received", "Bytes received from clients", varLabels, constLabels),
			"sent":          newServerZoneMetric(namespace, "sent", "Bytes sent to clients", varLabels, constLabels),
		},
		streamServerZoneMetrics: map[string]*prometheus.Desc{
			"processing":   newStreamServerZoneMetric(namespace, "processing", "Client connections that are currently being processed", varLabels, constLabels),
			"connections":  newStreamServerZoneMetric(namespace, "connections", "Total connections", varLabels, constLabels),
			"sessions_2xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"sessions_4xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"sessions_5xx": newStreamServerZoneMetric(namespace, "sessions", "Total sessions completed", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"received":     newStreamServerZoneMetric(namespace, "received", "Bytes received from clients", varLabels, constLabels),
			"sent":         newStreamServerZoneMetric(namespace, "sent", "Bytes sent to clients", varLabels, constLabels),
		},
		upstreamMetrics: map[string]*prometheus.Desc{
			"keepalives": newUpstreamMetric(namespace, "keepalives", "Idle keepalive connections", constLabels),
			"zombies":    newUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client requests", constLabels),
		},
		streamUpstreamMetrics: map[string]*prometheus.Desc{
			"zombies": newStreamUpstreamMetric(namespace, "zombies", "Servers removed from the group but still processing active client connections", constLabels),
		},
		upstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newUpstreamServerMetric(namespace, "state", "Current state", varLabels, constLabels),
			"active":                  newUpstreamServerMetric(namespace, "active", "Active connections", varLabels, constLabels),
			"requests":                newUpstreamServerMetric(namespace, "requests", "Total client requests", varLabels, constLabels),
			"responses_1xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx":           newUpstreamServerMetric(namespace, "responses", "Total responses sent to clients", varLabels, MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"sent":                    newUpstreamServerMetric(namespace, "sent", "Bytes sent to this server", varLabels, constLabels),
			"received":                newUpstreamServerMetric(namespace, "received", "Bytes received to this server", varLabels, constLabels),
			"fails":                   newUpstreamServerMetric(namespace, "fails", "Active connections", varLabels, constLabels),
			"unavail":                 newUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client requests (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold", varLabels, constLabels),
			"header_time":             newUpstreamServerMetric(namespace, "header_time", "Average time to get the response header from the server", varLabels, constLabels),
			"response_time":           newUpstreamServerMetric(namespace, "response_time", "Average time to get the full response from the server", varLabels, constLabels),
			"health_checks_checks":    newUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests", varLabels, constLabels),
			"health_checks_fails":     newUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks", varLabels, constLabels),
			"health_checks_unhealthy": newUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')", varLabels, constLabels),
		},
		streamUpstreamServerMetrics: map[string]*prometheus.Desc{
			"state":                   newStreamUpstreamServerMetric(namespace, "state", "Current state", varLabels, constLabels),
			"active":                  newStreamUpstreamServerMetric(namespace, "active", "Active connections", varLabels, constLabels),
			"sent":                    newStreamUpstreamServerMetric(namespace, "sent", "Bytes sent to this server", varLabels, constLabels),
			"received":                newStreamUpstreamServerMetric(namespace, "received", "Bytes received from this server", varLabels, constLabels),
			"fails":                   newStreamUpstreamServerMetric(namespace, "fails", "Number of unsuccessful attempts to communicate with the server", varLabels, constLabels),
			"unavail":                 newStreamUpstreamServerMetric(namespace, "unavail", "How many times the server became unavailable for client connections (state 'unavail') due to the number of unsuccessful attempts reaching the max_fails threshold", varLabels, constLabels),
			"connections":             newStreamUpstreamServerMetric(namespace, "connections", "Total number of client connections forwarded to this server", varLabels, constLabels),
			"connect_time":            newStreamUpstreamServerMetric(namespace, "connect_time", "Average time to connect to the upstream server", varLabels, constLabels),
			"first_byte_time":         newStreamUpstreamServerMetric(namespace, "first_byte_time", "Average time to receive the first byte of data", varLabels, constLabels),
			"response_time":           newStreamUpstreamServerMetric(namespace, "response_time", "Average time to receive the last byte of data", varLabels, constLabels),
			"health_checks_checks":    newStreamUpstreamServerMetric(namespace, "health_checks_checks", "Total health check requests", varLabels, constLabels),
			"health_checks_fails":     newStreamUpstreamServerMetric(namespace, "health_checks_fails", "Failed health checks", varLabels, constLabels),
			"health_checks_unhealthy": newStreamUpstreamServerMetric(namespace, "health_checks_unhealthy", "How many times the server became unhealthy (state 'unhealthy')", varLabels, constLabels),
		},
		streamZoneSyncMetrics: map[string]*prometheus.Desc{
			"bytes_in":        newStreamZoneSyncMetric(namespace, "bytes_in", "Bytes received by this node", constLabels),
			"bytes_out":       newStreamZoneSyncMetric(namespace, "bytes_out", "Bytes sent by this node", constLabels),
			"msgs_in":         newStreamZoneSyncMetric(namespace, "msgs_in", "Total messages received by this node", constLabels),
			"msgs_out":        newStreamZoneSyncMetric(namespace, "msgs_out", "Total messages sent by this node", constLabels),
			"nodes_online":    newStreamZoneSyncMetric(namespace, "nodes_online", "Number of peers this node is connected to", constLabels),
			"records_pending": newStreamZoneSyncZoneMetric(namespace, "records_pending", "The number of records that need to be sent to the cluster", constLabels),
			"records_total":   newStreamZoneSyncZoneMetric(namespace, "records_total", "The total number of records stored in the shared memory zone", constLabels),
		},
		locationZoneMetrics: map[string]*prometheus.Desc{
			"requests":      newLocationZoneMetric(namespace, "requests", "Total client requests", constLabels),
			"responses_1xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "1xx"})),
			"responses_2xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "2xx"})),
			"responses_3xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "3xx"})),
			"responses_4xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "4xx"})),
			"responses_5xx": newLocationZoneMetric(namespace, "responses", "Total responses sent to clients", MergeLabels(constLabels, prometheus.Labels{"code": "5xx"})),
			"discarded":     newLocationZoneMetric(namespace, "discarded", "Requests completed without sending a response", constLabels),
			"received":      newLocationZoneMetric(namespace, "received", "Bytes received from clients", constLabels),
			"sent":          newLocationZoneMetric(namespace, "sent", "Bytes sent to clients", constLabels),
		},
		resolverMetrics: map[string]*prometheus.Desc{
			"name":     newResolverMetric(namespace, "name", "Total requests to resolve names to addresses", constLabels),
			"srv":      newResolverMetric(namespace, "srv", "Total requests to resolve SRV records", constLabels),
			"addr":     newResolverMetric(namespace, "addr", "Total requests to resolve addresses to names", constLabels),
			"noerror":  newResolverMetric(namespace, "noerror", "Total number of successful responses", constLabels),
			"formerr":  newResolverMetric(namespace, "formerr", "Total number of FORMERR responses", constLabels),
			"servfail": newResolverMetric(namespace, "servfail", "Total number of SERVFAIL responses", constLabels),
			"nxdomain": newResolverMetric(namespace, "nxdomain", "Total number of NXDOMAIN responses", constLabels),
			"notimp":   newResolverMetric(namespace, "notimp", "Total number of NOTIMP responses", constLabels),
			"refused":  newResolverMetric(namespace, "refused", "Total number of REFUSED responses", constLabels),
			"timedout": newResolverMetric(namespace, "timedout", "Total number of timed out requests", constLabels),
			"unknown":  newResolverMetric(namespace, "unknown", "Total requests completed with an unknown error", constLabels),
		},
		upMetric: newUpMetric(namespace, constLabels),
		ICLabels: ICLabels{},
	}
}

// UpdateICLabels updates the labels with Ingress Controller related labels.
func (c *NginxPlusCollector) UpdateICLabels(ingLabels, vsLabels map[string]string) {
	c.ICLabels.serverZoneLabels = MergeLabels(ingLabels, vsLabels)
	c.ICLabels.upstreamServerLabels = MergeLabels(ingLabels, vsLabels)
	// log.Printf("DEBUG: Final labels %v", c.ICLabels.upstreamServerLabels)

	// Remove keys not present in upstream map.
	// stats, err := c.nginxClient.GetStats()
	// if err != nil {
	// 	log.Panicf("Could not get stats: %v", err)
	// }
	// for l := range c.ICLabels.upstreamServerLabels {
	// 	exists := false
	// 	for name := range stats.Upstreams {
	// 		log.Printf("DEBUG: %v, %v", l, name)
	// 		if name == l {
	// 			exists = true
	// 			break
	// 		}
	// 	}
	// 	if exists {
	// 		log.Printf("DEBUG: Label %v exists", l)
	// 	} else {
	// 		log.Printf("DEBUG: Removing label %v", l)
	// 		delete(c.ICLabels.upstreamServerLabels, l)
	// 	}
	// }
}

// Describe sends the super-set of all possible descriptors of NGINX Plus metrics
// to the provided channel.
func (c *NginxPlusCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.upMetric.Desc()

	for _, m := range c.totalMetrics {
		ch <- m
	}
	for _, m := range c.serverZoneMetrics {
		ch <- m
	}
	for _, m := range c.upstreamMetrics {
		ch <- m
	}
	for _, m := range c.upstreamServerMetrics {
		ch <- m
	}
	for _, m := range c.streamServerZoneMetrics {
		ch <- m
	}
	for _, m := range c.streamUpstreamMetrics {
		ch <- m
	}
	for _, m := range c.streamUpstreamServerMetrics {
		ch <- m
	}
	for _, m := range c.streamZoneSyncMetrics {
		ch <- m
	}
	for _, m := range c.locationZoneMetrics {
		ch <- m
	}
	for _, m := range c.resolverMetrics {
		ch <- m
	}
}

// Collect fetches metrics from NGINX Plus and sends them to the provided channel.
func (c *NginxPlusCollector) Collect(ch chan<- prometheus.Metric) {
	c.mutex.Lock() // To protect metrics from concurrent collects
	defer c.mutex.Unlock()

	stats, err := c.nginxClient.GetStats()
	if err != nil {
		c.upMetric.Set(nginxDown)
		ch <- c.upMetric
		log.Printf("Error getting stats: %v", err)
		return
	}

	c.upMetric.Set(nginxUp)
	ch <- c.upMetric

	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_accepted"],
		prometheus.CounterValue, float64(stats.Connections.Accepted))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_dropped"],
		prometheus.CounterValue, float64(stats.Connections.Dropped))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_active"],
		prometheus.GaugeValue, float64(stats.Connections.Active))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["connections_idle"],
		prometheus.GaugeValue, float64(stats.Connections.Idle))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["http_requests_total"],
		prometheus.CounterValue, float64(stats.HTTPRequests.Total))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["http_requests_current"],
		prometheus.GaugeValue, float64(stats.HTTPRequests.Current))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_handshakes"],
		prometheus.CounterValue, float64(stats.SSL.Handshakes))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_handshakes_failed"],
		prometheus.CounterValue, float64(stats.SSL.HandshakesFailed))
	ch <- prometheus.MustNewConstMetric(c.totalMetrics["ssl_session_reuses"],
		prometheus.CounterValue, float64(stats.SSL.SessionReuses))

	for name, zone := range stats.ServerZones {
		a := c.ICLabels.serverZoneLabels[name]
		variableLabels := MergeLabelList([]string{name}, []string{a})

		log.Printf("Name %v Lab %v", name, variableLabels)

		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["requests"],
			prometheus.CounterValue, float64(zone.Requests), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_1xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses1xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_2xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses2xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_3xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses3xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_4xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses4xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["responses_5xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses5xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.serverZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), variableLabels...)
	}

	for name, zone := range stats.StreamServerZones {
		a := c.ICLabels.serverZoneLabels[name]
		variableLabels := MergeLabelList([]string{name}, []string{a})

		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["processing"],
			prometheus.GaugeValue, float64(zone.Processing), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["connections"],
			prometheus.CounterValue, float64(zone.Connections), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_2xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions2xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_4xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions4xx), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sessions_5xx"],
			prometheus.CounterValue, float64(zone.Sessions.Sessions5xx), variableLabels...)
		// ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["discarded"],
		// 	prometheus.CounterValue, float64(zone.Discarded), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), variableLabels...)
		ch <- prometheus.MustNewConstMetric(c.streamServerZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), variableLabels...)
	}

	for name, upstream := range stats.Upstreams {
		for _, peer := range upstream.Peers {
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["requests"],
				prometheus.CounterValue, float64(peer.Requests), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_1xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses1xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_2xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses2xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_3xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses3xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_4xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses4xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["responses_5xx"],
				prometheus.CounterValue, float64(peer.Responses.Responses5xx), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["header_time"],
				prometheus.GaugeValue, float64(peer.HeaderTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), name, peer.Server)

			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.upstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), name, peer.Server)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["keepalives"],
			prometheus.GaugeValue, float64(upstream.Keepalives), name)
		ch <- prometheus.MustNewConstMetric(c.upstreamMetrics["zombies"],
			prometheus.GaugeValue, float64(upstream.Zombies), name)
	}

	for name, upstream := range stats.StreamUpstreams {
		for _, peer := range upstream.Peers {
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["state"],
				prometheus.GaugeValue, upstreamServerStates[peer.State], name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["active"],
				prometheus.GaugeValue, float64(peer.Active), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connections"],
				prometheus.CounterValue, float64(peer.Connections), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["connect_time"],
				prometheus.GaugeValue, float64(peer.ConnectTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["first_byte_time"],
				prometheus.GaugeValue, float64(peer.FirstByteTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["response_time"],
				prometheus.GaugeValue, float64(peer.ResponseTime), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["sent"],
				prometheus.CounterValue, float64(peer.Sent), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["received"],
				prometheus.CounterValue, float64(peer.Received), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["fails"],
				prometheus.CounterValue, float64(peer.Fails), name, peer.Server)
			ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["unavail"],
				prometheus.CounterValue, float64(peer.Unavail), name, peer.Server)
			if peer.HealthChecks != (plusclient.HealthChecks{}) {
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_checks"],
					prometheus.CounterValue, float64(peer.HealthChecks.Checks), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_fails"],
					prometheus.CounterValue, float64(peer.HealthChecks.Fails), name, peer.Server)
				ch <- prometheus.MustNewConstMetric(c.streamUpstreamServerMetrics["health_checks_unhealthy"],
					prometheus.CounterValue, float64(peer.HealthChecks.Unhealthy), name, peer.Server)
			}
		}
		ch <- prometheus.MustNewConstMetric(c.streamUpstreamMetrics["zombies"],
			prometheus.GaugeValue, float64(upstream.Zombies), name)
	}

	if stats.StreamZoneSync != nil {
		for name, zone := range stats.StreamZoneSync.Zones {
			ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["records_pending"],
				prometheus.GaugeValue, float64(zone.RecordsPending), name)
			ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["records_total"],
				prometheus.GaugeValue, float64(zone.RecordsTotal), name)
		}

		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["bytes_in"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.BytesIn))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["bytes_out"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.BytesOut))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["msgs_in"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.MsgsIn))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["msgs_out"],
			prometheus.CounterValue, float64(stats.StreamZoneSync.Status.MsgsOut))
		ch <- prometheus.MustNewConstMetric(c.streamZoneSyncMetrics["nodes_online"],
			prometheus.GaugeValue, float64(stats.StreamZoneSync.Status.NodesOnline))
	}

	for name, zone := range stats.LocationZones {
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["requests"],
			prometheus.CounterValue, float64(zone.Requests), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_1xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses1xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_2xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses2xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_3xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses3xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_4xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses4xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["responses_5xx"],
			prometheus.CounterValue, float64(zone.Responses.Responses5xx), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["discarded"],
			prometheus.CounterValue, float64(zone.Discarded), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["received"],
			prometheus.CounterValue, float64(zone.Received), name)
		ch <- prometheus.MustNewConstMetric(c.locationZoneMetrics["sent"],
			prometheus.CounterValue, float64(zone.Sent), name)
	}

	for name, zone := range stats.Resolvers {
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["name"],
			prometheus.CounterValue, float64(zone.Requests.Name), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["srv"],
			prometheus.CounterValue, float64(zone.Requests.Srv), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["addr"],
			prometheus.CounterValue, float64(zone.Requests.Addr), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["noerror"],
			prometheus.CounterValue, float64(zone.Responses.Noerror), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["formerr"],
			prometheus.CounterValue, float64(zone.Responses.Formerr), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["servfail"],
			prometheus.CounterValue, float64(zone.Responses.Servfail), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["nxdomain"],
			prometheus.CounterValue, float64(zone.Responses.Nxdomain), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["notimp"],
			prometheus.CounterValue, float64(zone.Responses.Notimp), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["refused"],
			prometheus.CounterValue, float64(zone.Responses.Refused), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["timedout"],
			prometheus.CounterValue, float64(zone.Responses.Timedout), name)
		ch <- prometheus.MustNewConstMetric(c.resolverMetrics["unknown"],
			prometheus.CounterValue, float64(zone.Responses.Unknown), name)
	}
}

var upstreamServerStates = map[string]float64{
	"up":        1.0,
	"draining":  2.0,
	"down":      3.0,
	"unavail":   4.0,
	"checking":  5.0,
	"unhealthy": 6.0,
}

// TODO: Label cardinality issue. E.g if podname is not set. Or running as standalone exporter?
func newServerZoneMetric(namespace string, metricName string, docString string, variableLabels VariableLabels, constLabels prometheus.Labels) *prometheus.Desc {
	var labels = []string{"server_zone"}
	labels = MergeLabelList(labels, variableLabels.ServerZoneLabels)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "server_zone", metricName), docString, labels, constLabels)
}

func newStreamServerZoneMetric(namespace string, metricName string, docString string, variableLabels VariableLabels, constLabels prometheus.Labels) *prometheus.Desc {
	var labels = []string{"server_zone"}
	labels = MergeLabelList(labels, variableLabels.ServerZoneLabels)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_server_zone", metricName), docString, labels, constLabels)
}

func newUpstreamMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream", metricName), docString, []string{"upstream"}, constLabels)
}

func newStreamUpstreamMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream", metricName), docString, []string{"upstream"}, constLabels)
}

func newUpstreamServerMetric(namespace string, metricName string, docString string, variableLabels VariableLabels, constLabels prometheus.Labels) *prometheus.Desc {
	var labels = []string{"upstream", "server"}
	labels = MergeLabelList(labels, variableLabels.UpstreamServerLabels)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "upstream_server", metricName), docString, labels, constLabels)
}

func newStreamUpstreamServerMetric(namespace string, metricName string, docString string, variableLabels VariableLabels, constLabels prometheus.Labels) *prometheus.Desc {
	var labels = []string{"upstream", "server"}
	labels = MergeLabelList(labels, variableLabels.UpstreamServerLabels)
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_upstream_server", metricName), docString, labels, constLabels)
}

func newStreamZoneSyncMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_status", metricName), docString, nil, constLabels)
}

func newStreamZoneSyncZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "stream_zone_sync_zone", metricName), docString, []string{"zone"}, constLabels)
}

func newLocationZoneMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "location_zone", metricName), docString, []string{"location_zone"}, constLabels)
}

func newResolverMetric(namespace string, metricName string, docString string, constLabels prometheus.Labels) *prometheus.Desc {
	return prometheus.NewDesc(prometheus.BuildFQName(namespace, "resolver", metricName), docString, []string{"resolver"}, constLabels)
}
