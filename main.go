package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	analyticsdata "google.golang.org/api/analyticsdata/v1beta"
	"google.golang.org/api/analyticsreporting/v4"
	"google.golang.org/api/option"
)

var (
	analyticsSvc *analyticsreporting.Service
)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	ctx := context.Background()

	initAnalytics(ctx)

	http.HandleFunc("/report", getReportHandler)
	http.HandleFunc("/customReport", getCustomReportHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("Server listening on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func initAnalytics(ctx context.Context) {
	var err error
	credPath := os.Getenv("GA_CREDENTIALS_PATH")
	if credPath == "" {
		log.Fatal("GA_CREDENTIALS_PATH environment variable is required")
	}
	analyticsSvc, err = analyticsreporting.NewService(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		log.Fatalf("Failed to create Analytics service: %v", err)
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
		log.Fatal("GA_VIEW_ID environment variable is required")
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
	ctx := r.Context()

	credPath := os.Getenv("GA_CREDENTIALS_PATH")
	if credPath == "" {
		log.Fatal("GA_CREDENTIALS_PATH environment variable is required")
	}

	client, err := analyticsdata.NewService(ctx, option.WithCredentialsFile(credPath))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	propertyID := os.Getenv("GA_PROPERTY_ID")
	if propertyID == "" {
		log.Fatal("GA_PROPERTY_ID environment variable is required")
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

	resp, err := client.Properties.RunReport("properties/"+propertyID, req).Do()
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
