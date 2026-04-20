package browsertester

import rt "browsertester/internal/runtime"

type GeolocationPosition = rt.GeolocationPosition

const (
	GeolocationErrorPermissionDenied   = rt.GeolocationErrorPermissionDenied
	GeolocationErrorPositionUnavailable = rt.GeolocationErrorPositionUnavailable
	GeolocationErrorTimeout            = rt.GeolocationErrorTimeout
)

type GeolocationView struct {
	session *rt.Session
}

func (h *Harness) Geolocation() GeolocationView {
	if h == nil || h.session == nil {
		return GeolocationView{}
	}
	return GeolocationView{session: h.session}
}

func (v GeolocationView) EmitPosition(position GeolocationPosition) error {
	if v.session == nil {
		return NewError(ErrorKindMock, "geolocation is unavailable")
	}
	if err := v.session.EmitGeolocationPosition(position); err != nil {
		return NewError(ErrorKindMock, err.Error())
	}
	return nil
}

func (v GeolocationView) EmitError(code int, message string) error {
	if v.session == nil {
		return NewError(ErrorKindMock, "geolocation is unavailable")
	}
	if err := v.session.EmitGeolocationError(code, message); err != nil {
		return NewError(ErrorKindMock, err.Error())
	}
	return nil
}
