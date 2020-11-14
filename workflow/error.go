package workflow

// DockerError occures when something went wrong with docker
// and we just want to pass back the original error, but with
// a fancy wrap.
// Most of them will be converted to other error types, but
// if any of them left in there, it will be a DockerError.
type DockerError struct {
	Original error
}

func (e DockerError) Error() string {
	return e.Original.Error()
}
