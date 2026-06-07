package mcpregistry

// ZIMEntry describes a single ZIM archive available for local documentation.
type ZIMEntry struct {
	Slug         string
	DisplayName  string
	URL          string
	ExpectedHash string
	SizeBytes    int64
	Ecosystems   []string
}

// BuiltinZIMCatalog lists the ZIM archives that qsdev can download for
// offline Stack Exchange documentation.
var BuiltinZIMCatalog = []ZIMEntry{
	{
		Slug:         "unix.stackexchange.com_en_all_2025-06",
		DisplayName:  "Unix & Linux Stack Exchange",
		URL:          "https://download.kiwix.org/zim/stack_exchange/unix.stackexchange.com_en_all_2025-06.zim",
		ExpectedHash: "", // populated when hash is known
		SizeBytes:    850_000_000,
		Ecosystems:   []string{"go", "python", "rust", "javascript"},
	},
	{
		Slug:         "serverfault.com_en_all_2025-06",
		DisplayName:  "Server Fault",
		URL:          "https://download.kiwix.org/zim/stack_exchange/serverfault.com_en_all_2025-06.zim",
		ExpectedHash: "",
		SizeBytes:    750_000_000,
		Ecosystems:   []string{"go", "python", "rust", "javascript"},
	},
	{
		Slug:         "softwareengineering.stackexchange.com_en_all_2025-06",
		DisplayName:  "Software Engineering SE",
		URL:          "https://download.kiwix.org/zim/stack_exchange/softwareengineering.stackexchange.com_en_all_2025-06.zim",
		ExpectedHash: "",
		SizeBytes:    450_000_000,
		Ecosystems:   []string{"go", "python", "rust", "javascript"},
	},
	{
		Slug:         "devops.stackexchange.com_en_all_2025-06",
		DisplayName:  "DevOps SE",
		URL:          "https://download.kiwix.org/zim/stack_exchange/devops.stackexchange.com_en_all_2025-06.zim",
		ExpectedHash: "",
		SizeBytes:    200_000_000,
		Ecosystems:   []string{"go", "python", "rust", "javascript"},
	},
	{
		Slug:         "dba.stackexchange.com_en_all_2025-06",
		DisplayName:  "DBA SE",
		URL:          "https://download.kiwix.org/zim/stack_exchange/dba.stackexchange.com_en_all_2025-06.zim",
		ExpectedHash: "",
		SizeBytes:    400_000_000,
		Ecosystems:   []string{"go", "python", "rust", "javascript"},
	},
}
