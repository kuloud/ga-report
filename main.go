package handler

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/analyticsreporting/v4"
	"google.golang.org/api/option"
)

var (
	analyticsSvc     *analyticsreporting.Service
	analyticsDataSvc *analyticsdata.Service
)

const (
	// Define the required scope for Analytics APIs
	analyticsScope = "https://www.googleapis.com/auth/analytics.readonly"
)

func init() {
	// if err := godotenv.Load(); err != nil {
	// 	log.Print("No .env file found")
	// }

	ctx := context.Background()
	initAnalytics(ctx)
}

func Handler(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/report":
		getReportHandler(w, r)
	case "/customReport":
		getCustomReportHandler(w, r)
	default:
		http.NotFound(w, r)
	}
}

func initAnalytics(ctx context.Context) {
	gac := os.Getenv("GA_CREDENTIALS")
	if gac == "" {
		log.Fatal("GA_CREDENTIALS environment variable is required")
	}
	log.Printf("GA_CREDENTIALS: %s", gac)

	var err error
	// Initialize Analytics Reporting API (v4) for Universal Analytics
	analyticsSvc, err = analyticsreporting.NewService(ctx, option.WithCredentialsJSON([]byte(gac)), option.WithScopes(analyticsScope))
	if err != nil {
		log.Fatalf("Failed to create Analytics Reporting service: %v", err)
	}

	// Initialize Analytics Data API (v1beta) for GA4
	analyticsDataSvc, err = analyticsdata.NewService(ctx, option.WithCredentialsJSON([]byte(gac)), option.WithScopes(analyticsScope))
	if err != nil {
		log.Fatalf("Failed to create Analytics Data service: %v", err)
	}
}

func getReportHandler(w http.ResponseWriter, r *http.Request) {
	report, err := getAnalyticsReport(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(report))
}

func getAnalyticsReport(ctx context.Context) (string, error) {
	viewId := os.Getenv("GA_VIEW_ID")
	if viewId == "" {
		return "", http.ErrMissingFile // Avoid log.Fatal in a handler
	}
	log.Printf("Fetching report for view ID: %s", viewId)

	req := &analyticsreporting.GetReportsRequest{
		ReportRequests: []*analyticsreporting.ReportRequest{
			{
				ViewId: viewId,
				DateRanges: []*analyticsreporting.DateRange{
					{StartDate: "7daysAgo", EndDate: "today"},
				},
				Metrics: []*analyticsreporting.Metric{
					{Expression: "ga:sessions"},
					{Expression: "ga:users"},
				},
			},
		},
	}

	resp, err := analyticsSvc.Reports.BatchGet(req).Do()
	if err != nil {
		log.Printf("Error fetching report: %v", err)
		return "", err
	}

	jsonData, err := json.Marshal(resp)
	if err != nil {
		return "", err
	}
	log.Printf("Fetched report: %s", string(jsonData))

	return string(jsonData), nil
}

func getCustomReportHandler(w http.ResponseWriter, r *http.Request) {

	propertyID := os.Getenv("GA_PROPERTY_ID")
	if propertyID == "" {
		http.Error(w, "GA_PROPERTY_ID environment variable is required", http.StatusInternalServerError)
		return
	}

	req := &analyticsdata.RunReportRequest{
		Property: "properties/" + propertyID,
		DateRanges: []*analyticsdata.DateRange{
			{StartDate: "2025-03-01", EndDate: "2025-04-01"},
		},
		Dimensions: []*analyticsdata.Dimension{
			{Name: "date"},
			{Name: "country"},
		},
		Metrics: []*analyticsdata.Metric{
			{Name: "activeUsers"},
			{Name: "screenPageViews"},
		},
	}

	resp, err := analyticsDataSvc.Properties.RunReport("properties/"+propertyID, req).Do()
	if err != nil {
		log.Printf("Error fetching GA4 report: %v", err)
		http.Error(w, "Failed to fetch report: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("Error encoding response: %v", err)
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	log.Printf("Fetched GA4 report for property %s", propertyID)
}
