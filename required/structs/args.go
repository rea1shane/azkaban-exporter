package structs

type Args struct {
	ListenAddress          string
	MetricsPath            string
	DisableExporterMetrics bool
	MaxRequests            int
}
