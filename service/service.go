package service

type T interface {
	start()
	work()
	stop()
	handle() Handle
	deps() []T
}

func Start(svc T) {
	h := svc.handle()
	if h.IsActive() {
		return
	}

	for _, dep := range svc.deps() {
		if dep != nil {
			Start(dep)
		}
	}

	work := h.start()
	svc.start()
	go func() {
		defer func() {
			svc.stop()
			h.stopped()
		}()

		for {
			select {
			case <-stop:
				return

			case <-work:
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
