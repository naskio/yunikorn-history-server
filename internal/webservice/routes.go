package webservice

import (
	"context"
	"net/http"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/rs/cors"

	"github.com/G-Research/yunikorn-history-server/internal/log"
)

const (
	// routes
	routeClusters                 = "/ws/v1/clusters"
	routePartitions               = "/ws/v1/partitions"
	routeQueuesPerPartition       = "/ws/v1/partition/:partition_name/queues"
	routeAppsPerPartitionPerQueue = "/ws/v1/partition/:partition_name/queue/:queue_name/applications"
	routeAppsHistory              = "/ws/v1/history/apps"
	routeContainersHistory        = "/ws/v1/history/containers"
	routeNodesPerPartition        = "/ws/v1/partition/:partition_name/nodes"
	routeNodeUtilization          = "/ws/v1/scheduler/node-utilizations"
	routeSchedulerHealthcheck     = "/ws/v1/scheduler/healthcheck"
	routeEventStatistics          = "/ws/v1/event-statistics"
	routeHealthLiveness           = "/ws/v1/health/liveness"
	routeHealthReadiness          = "/ws/v1/health/readiness"

	// params
	paramsPartitionName = "partition_name"
	paramsQueueName     = "queue_name"
)

func (ws *WebService) init(ctx context.Context) {
	router := httprouter.New()
	router.NotFound = http.HandlerFunc(ws.serveSPA)

	router.Handle(http.MethodGet, routePartitions, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getPartitions(w, r, p)
	})
	router.Handle(http.MethodGet, routeQueuesPerPartition, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getQueuesPerPartition(w, r, p)
	})
	router.Handle(http.MethodGet, routeAppsPerPartitionPerQueue, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getAppsPerPartitionPerQueue(w, r, p)
	})
	router.Handle(http.MethodGet, routeNodesPerPartition, func(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getNodesPerPartition(w, r, p)
	})
	router.Handle(http.MethodGet, routeAppsHistory, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getAppsHistory(w, r)
	})
	router.Handle(http.MethodGet, routeContainersHistory, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getContainersHistory(w, r)
	})
	router.Handle(http.MethodGet, routeNodeUtilization, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getNodeUtilizations(w, r)
	})
	router.Handle(http.MethodGet, routeEventStatistics, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.getEventStatistics(w, r)
	})
	router.Handle(http.MethodGet, routeHealthLiveness, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.LivenessHealthcheck(w, r)
	})
	router.Handle(http.MethodGet, routeHealthReadiness, func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		enrichRequestContext(ctx, r)
		ws.ReadinessHealthcheck(w, r)
	})

	// Setup CORS
	c := cors.New(ws.corsConfig)
	ws.server.Handler = c.Handler(router)
}

func enrichRequestContext(ctx context.Context, r *http.Request) {
	logger := log.FromContext(ctx)
	rid := uuid.New().String()
	logger = logger.With("request_id", rid)
	ctx = log.ToContext(ctx, logger)
	*r = *r.WithContext(ctx)
}

func (ws *WebService) getPartitions(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	partitions, err := ws.repository.GetAllPartitions(r.Context())
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, partitions)
}

func (ws *WebService) getQueuesPerPartition(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	partition := params.ByName(paramsPartitionName)
	queues, err := ws.repository.GetQueuesPerPartition(r.Context(), partition)
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, queues)
}

// getAppsPerPartitionPerQueue returns all applications for a given partition and queue.
// Results are ordered by submission time in descending order.
// Following query params are supported:
// - user: filter by user
// - groups: filter by groups (comma-separated list)
// - submissionStartTime: filter from the submission time
// - submissionEndTime: filter until the submission time
// - limit: limit the number of returned applications
// - offset: offset the returned applications
func (ws *WebService) getAppsPerPartitionPerQueue(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	partition := params.ByName(paramsPartitionName)
	queue := params.ByName(paramsQueueName)

	filters, err := parseApplicationFilters(r)
	if err != nil {
		badRequestResponse(w, r, err)
		return
	}

	apps, err := ws.repository.GetAppsPerPartitionPerQueue(r.Context(), partition, queue, *filters)
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, apps)
}

func (ws *WebService) getNodesPerPartition(w http.ResponseWriter, r *http.Request, params httprouter.Params) {
	partition := params.ByName(paramsPartitionName)
	nodes, err := ws.repository.GetNodesPerPartition(r.Context(), partition)
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, nodes)
}

func (ws *WebService) getAppsHistory(w http.ResponseWriter, r *http.Request) {
	appsHistory, err := ws.repository.GetApplicationsHistory(r.Context())
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, appsHistory)
}

func (ws *WebService) getContainersHistory(w http.ResponseWriter, r *http.Request) {
	containersHistory, err := ws.repository.GetContainersHistory(r.Context())
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, containersHistory)
}

func (ws *WebService) getNodeUtilizations(w http.ResponseWriter, r *http.Request) {
	nodeUtilization, err := ws.repository.GetNodeUtilizations(r.Context())
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, nodeUtilization)
}

func (ws *WebService) getEventStatistics(w http.ResponseWriter, r *http.Request) {
	counts, err := ws.eventRepository.Counts(r.Context())
	if err != nil {
		errorResponse(w, r, err)
		return
	}
	jsonResponse(w, counts)
}

func (ws *WebService) LivenessHealthcheck(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, ws.healthService.Liveness(r.Context()))
}

func (ws *WebService) ReadinessHealthcheck(w http.ResponseWriter, r *http.Request) {
	jsonResponse(w, ws.healthService.Readiness(r.Context()))
}

func (ws *WebService) serveSPA(w http.ResponseWriter, r *http.Request) {
	path := filepath.Join(ws.assetsDir, r.URL.Path)
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		http.ServeFile(w, r, filepath.Join(ws.assetsDir, "index.html"))
		return
	}

	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.FileServer(http.Dir(ws.assetsDir)).ServeHTTP(w, r)
}
