package builder

import "sync"

var (
	pathTemplateCache   = make(map[string]*pathTemplate)
	pathTemplateCacheMu sync.RWMutex
)

func getCachedPathTemplate(path string) (*pathTemplate, bool) {
	pathTemplateCacheMu.RLock()
	defer pathTemplateCacheMu.RUnlock()
	tmpl, ok := pathTemplateCache[path]
	return tmpl, ok
}

func setPathTemplateCache(path string, tmpl *pathTemplate) {
	pathTemplateCacheMu.Lock()
	defer pathTemplateCacheMu.Unlock()
	pathTemplateCache[path] = tmpl
}
