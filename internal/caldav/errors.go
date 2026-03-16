package caldav

import "errors"

var ErrPreconditionFailed = errors.New("caldav precondition failed")
var ErrMissingETag = errors.New("caldav missing etag precondition")
var ErrInvalidTaskHref = errors.New("caldav invalid task href")
