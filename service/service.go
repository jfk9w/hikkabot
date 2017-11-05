package service

type T interface {
	start()
	work()
	stop()

	handle() *Handle
	deps() []T
}

func Start(svc T) {
	h := svc.handle()
	if h.IsActive() {
		return
	}

	for _, dep := range svc.deps() {
		Start(dep)
	}

	stop := h.start()
	svc.start()
	go func() {
		defer func() {
			svc.stop()
			h.notify()
		}()

		for {
			select {
			case <-stop:
				return

			default:
				svc.work()
			}
		}
	}()
}

func Stop(svc T, sync bool) {
	h := svc.handle()
	if !h.IsActive() {
		return
	}

	for _, dep := range svc.deps() {
		Stop(dep, sync)
	}

	h.Stop()
	if sync {
		h.wait()
	}
}
