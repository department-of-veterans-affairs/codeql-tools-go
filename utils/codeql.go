package utils

var supportedCodeQLLanguages = []string{
	"c",
	"cpp",
	"csharp",
	"go",
	"java",
	"kotlin",
	"javascript",
	"python",
	"ruby",
	"typescript",
}

func IsSupportedCodeQLLanguage(language string) bool {
	for _, supportedLanguage := range supportedCodeQLLanguages {
		if language == supportedLanguage {
			return true
		}
	}
	return false
}
