package store

import "path"

const (
	Update = "UPDATE"
	Delete = "DELETE"
)

type Event struct {
	Action string `json:"action"`
	Path   string `json:"path"`
	Value  string `json:"value"`
}

func newEvent(action string, path string, value string) *Event {
	return &Event{
		Action: action,
		Path:   path,
		Value:  value,
	}
}

type Watcher interface {
	EventChan() chan *Event
	Remove()
}

type watcher struct {
	eventChan chan *Event
	removed   bool
	node      *node
	remove    func()
}

func newWatcher(node *node) *watcher {
	w := &watcher{
		eventChan: make(chan *Event, 10),
		node:      node,
	}
	return w
}

func (w *watcher) EventChan() chan *Event {
	return w.eventChan
}

func (w *watcher) Remove() {
	w.node.store.watcherLock.Lock()
	defer w.node.store.watcherLock.Unlock()

	close(w.eventChan)
	if w.remove != nil {
		w.remove()
	}
}

type aggregateWatcher struct {
	watchers  map[string]Watcher
	eventChan chan *Event
}

func NewAggregateWatcher(watchers map[string]Watcher) Watcher {
	eventChan := make(chan *Event, len(watchers))
	for pathPrefix, watcher := range watchers {
		go func() {
			for {
				select {
				case event, ok := <-watcher.EventChan():
					if ok {
						eventChan <- newEvent(event.Action, path.Join(pathPrefix, event.Path), event.Value)
					} else {
						return
					}
				}
			}
		}()
	}
	return &aggregateWatcher{watchers: watchers, eventChan: eventChan}
}

func (w *aggregateWatcher) EventChan() chan *Event {
	return w.eventChan
}

func (w *aggregateWatcher) Remove() {
	for _, watcher := range w.watchers {
		watcher.Remove()
	}
	close(w.eventChan)
}
