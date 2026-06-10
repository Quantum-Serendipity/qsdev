package risk

type PercentileTable struct {
	DownloadsP10  int64
	DownloadsP50  int64
	DownloadsP90  int64
	DependentsP10 int
	DependentsP50 int
	DependentsP90 int
}

var EcosystemPercentiles = map[Ecosystem]PercentileTable{
	EcosystemNpm: {
		DownloadsP10:  50,
		DownloadsP50:  5000,
		DownloadsP90:  500000,
		DependentsP10: 0,
		DependentsP50: 5,
		DependentsP90: 500,
	},
	EcosystemPyPI: {
		DownloadsP10:  30,
		DownloadsP50:  3000,
		DownloadsP90:  300000,
		DependentsP10: 0,
		DependentsP50: 3,
		DependentsP90: 200,
	},
	EcosystemGo: {
		DownloadsP10:  10,
		DownloadsP50:  1000,
		DownloadsP90:  100000,
		DependentsP10: 0,
		DependentsP50: 5,
		DependentsP90: 300,
	},
	EcosystemCargo: {
		DownloadsP10:  20,
		DownloadsP50:  2000,
		DownloadsP90:  200000,
		DependentsP10: 0,
		DependentsP50: 3,
		DependentsP90: 150,
	},
	EcosystemNuGet: {
		DownloadsP10:  40,
		DownloadsP50:  4000,
		DownloadsP90:  400000,
		DependentsP10: 0,
		DependentsP50: 4,
		DependentsP90: 250,
	},
	EcosystemRubyGems: {
		DownloadsP10:  25,
		DownloadsP50:  2500,
		DownloadsP90:  250000,
		DependentsP10: 0,
		DependentsP50: 3,
		DependentsP90: 180,
	},
}
