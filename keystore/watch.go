package keystore

import (
  "time"

  "github.com/rjeczalik/notify"
)

type watcher struct {
  ac       *accountCache
  starting bool
  running  bool
  ev       chan notify.EventInfo
  quit     chan struct{}
}

func newWatcher(ac *accountCache) *watcher {
  return &watcher{
    ac:   ac,
    ev:   make(chan notify.EventInfo, 10),
    quit: make(chan struct{}),
  }
}

func (w *watcher) start() {
  if w.starting || w.running {
    return
  }
  w.starting = true
  go w.loop()
}

func (w *watcher) close() {
  close(w.quit)
}

func (w *watcher) loop() {
  defer func() {
    w.ac.mu.Lock()
    w.running = false
    w.starting = false
    w.ac.mu.Unlock()
  }()

  if err := notify.Watch(w.ac.keydir, w.ev, notify.All); err != nil {
    return
  }
  defer notify.Stop(w.ev)

  // TODO: Log that started and stopped

  w.ac.mu.Lock()
  w.running = true
  w.ac.mu.Unlock()

  // Wait for file system events and reload.
  // When an event occurs, the reload call is delayed a bit so that
  // multiple events arriving quickly only cause a single reload.
  var (
    debounceDuration = 500 * time.Millisecond
    rescanTriggered  = false
    debounce         = time.NewTimer(0)
  )

  // Ignore initial trigger
  if !debounce.Stop() {
    <-debounce.C
  }
  defer debounce.Stop()
  for {
    select {
    case <-w.quit:
      return
    case <-w.ev:
      // Trigger the scan (with delay), if not already triggered
      if !rescanTriggered {
        debounce.Reset(debounceDuration)
        rescanTriggered = false
      }
    case <-debounce.C:
      w.ac.scanAccounts()
      rescanTriggered = false
    }
  }
}
