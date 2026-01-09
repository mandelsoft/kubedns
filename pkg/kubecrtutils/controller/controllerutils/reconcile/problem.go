package reconcile

import (
	"errors"
	"fmt"
	"time"

	"github.com/mandelsoft/goutils/general"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Problem interface {
	Requeue() bool
	Problem() error
	Error() error

	String() string
	Message() string
}

type problem struct {
	requeue bool
	problem error
	err     error
}

// Requeue requests a ratelimited back off without reporting a reconciliation error
// iff there is an error given.
func Requeue(err error) Problem {
	if err == nil {
		return nil
	}
	return problem{requeue: true, problem: err}
}

func Requeuef(msg string, args ...any) Problem {
	return Requeue(fmt.Errorf(msg, args...))
}

// Failed reports a persistent error which does not require a requeue
// iff there is an error given.
// It descibes an error on the resource which requires a modification
// by the maintainer to solve the problem.
func Failed(err error) Problem {
	return problem{err: err}
}

func Failedf(msg string, args ...any) Problem {
	return Failed(fmt.Errorf(msg, args...))
}

// TemporaryProblem reports a temporary error requiring a requeue and error reporting
// iff there is an error given.
// It describes a reconciliation error which probably disappears after repeating
// the request.
func TemporaryProblem(err error) Problem {
	if err == nil {
		return nil
	}
	return problem{err: err, requeue: true}
}

func TemporaryProblemf(msg string, args ...any) Problem {
	return TemporaryProblem(fmt.Errorf(msg, args...))
}

// WatchBackedProblem describes a problem without error reporting and requeue
// iff there is an error given.
// The requeue will be done by a secondary watch on the problematic resource.
func WatchBackedProblem(err error) Problem {
	if err == nil {
		return nil
	}
	return problem{problem: err}
}

func WatchBackedProblemf(msg string, args ...any) Problem {
	return WatchBackedProblem(fmt.Errorf(msg, args...))
}

func (p problem) Requeue() bool {
	return p.requeue
}
func (p problem) Problem() error {
	return p.problem
}
func (p problem) Error() error {
	return p.err
}

func (p problem) String() string {
	if p.err != nil {
		if p.problem != nil {
			return fmt.Sprintf("reconciliation error: %s; auto healing problem: %s", p.err.Error(), p.problem.Error())
		}
		return "reconciliation error: " + p.err.Error()
	}

	if p.problem != nil {
		if p.requeue {
			return "backoff problem: " + p.problem.Error()
		} else {
			return "triggerable problem: " + p.problem.Error()
		}
	}
	return "no reconciliation problem"
}

func (p problem) Message() string {
	if p.err != nil {
		if p.problem != nil {
			return fmt.Sprintf("%s; %s", p.err.Error(), p.problem.Error())
		}
		return p.err.Error()
	}

	if p.problem != nil {
		return p.problem.Error()
	}
	return "no problem"
}

////////////////////////////////////////////////////////////////////////////////

func AggregateProblem(ps ...Problem) Problem {
	if len(ps) == 0 {
		return nil
	}
	if len(ps) == 1 {
		return ps[0]
	}

	var result problem
	found := false
	for _, p := range ps {
		if p == nil {
			continue
		}
		found = true
		result.requeue = result.requeue || p.Requeue()
		result.err = errors.Join(result.err, p.Error())
		result.problem = errors.Join(result.problem, p.Problem())
	}

	if !found {
		return nil
	}
	return result
}

////////////////////////////////////////////////////////////////////////////////

type InfoLogger interface {
	Info(msg string, keysAndValues ...interface{})
}

// cases:
//  - reconciliation successful and completed
//  - reconciliation successful but incomplete (for example waiting for
//    secondary/external resources to read an expected state)
//  - reconciliation temporarily failed (for example due to API issues)
//  - reconciliation permanently failed (resource configuratiuon issue)

func Result(log InfoLogger, p Problem, after ...time.Duration) (ctrl.Result, error) {
	if p == nil {
		log.Info("*** reconciliation completed")
		return ctrl.Result{RequeueAfter: general.Optional(after...)}, nil
	}
	if p.Error() != nil {
		if p.Requeue() {
			log.Info("*** temporary reconciliation problem: {{error}}", "error", p.String())
			return ctrl.Result{}, p.Error()
		} else {
			if p.Problem() == nil {
				log.Info("*** permanent resource problem -> wait for resource modification: {{error}}", "error", p.String())
				return ctrl.Result{}, nil
			}
		}
	}

	result := ctrl.Result{
		Requeue:      p.Requeue(),
		RequeueAfter: general.Optional(after...),
	}

	log.Info("*** reconciliation incomplete: {{error}}", "error", p.String())
	var err error
	if p.Requeue() {
		// Requeue does not reliably work if watch close appears,
		// so, enforce an error.
		err = p.Problem()
	}
	return result, err
}
