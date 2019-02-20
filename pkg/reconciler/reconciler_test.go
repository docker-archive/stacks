package reconciler

import (
	// Ginkgo uses the dot-import for its packages. This may seem strange, but
	// the tests flow much better without having to qualify all of the Ginkgo
	// imports with package names.
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	// additionally, we're gonna custom-brew some matchers by composition, so
	// we'll need to return GomegaMatcher, which is in the types package
	. "github.com/onsi/gomega/types"

	// using errdefs makes handling errors across package boundaries actually
	// sensible
	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/interfaces"
)

const (
	stackID   = "someIDwhocares"
	stackName = "someNamewhocares"
)

// obj is just a tuple of 2 strings for use in the fakeObjectChangeNotifier
type obj struct {
	kind, id string
}

// fakeObjectChangeNotifier implements the ObjectChangeNotifier interface and
// keeps track of which objects it was notified about and in what order.
type fakeObjectChangeNotifier struct {
	objects []obj
}

func (f *fakeObjectChangeNotifier) Notify(kind, id string) {
	f.objects = append(f.objects, obj{kind, id})
}

// ConsistOfServices is a matcher that verifies that a map of services contains
// only services whose specs match the provided specs.
func ConsistOfServices(specs []swarm.ServiceSpec) GomegaMatcher {
	// quick function to convert the map to a slice of ServiceSpecs
	serviceSpecs := func(f *fakeReconcilerClient) []swarm.ServiceSpec {
		specs := make([]swarm.ServiceSpec, 0, len(f.services))
		for _, service := range f.services {
			specs = append(specs, service.Spec)
		}
		return specs
	}
	// Transform the actual value, and then ensure it consists of the
	// provided specs
	return WithTransform(serviceSpecs, ConsistOf(specs))
}

// WTF AM I LOOKING AT AND HOW DOES THIS WORK: A PRIMER ON GINKGO TESTS
//
// Ginkgo is a BDD framework. BDD means a lot of things that don't really
// matter to us and implies a whole software engineering methodology that we
// don't use. What _is_ important is that Ginkgo is a test framework that
// provides the closest thing to a DSL you can produce in golang, which allows
// us to describe tests in a natural-language-like method. I (@dperny) like
// this because my typical development pattern is to write comments first and
// code second, and this makes comments INTO code IMHO.
//
// Ginkgo uses a series of blocks to describe behaviors. "Describe", "Context",
// and "When" all specify overarching container blocks and are aliases for each
// other. "It" describes test cases. "BeforeEach" and "AfterEach" are setup and
// tear down, and "JustBeforeEach"/"JustAfterEach" are setup and teardown that
// run first/last.
//
// Ginkgo relies heavily on closures instead of function calls, so to share
// data between them, we need to use variables ("var" blocks). Typically, these
// variables are initialized in the BeforeEach or JustBeforeEach sections. A
// side effect of this reliance on variables is that ginkgo tests cannot
// execute concurrently.
//
// We also use "Gomega", which is a matcher/assertion library. This lets us
// describe the actual test cases in the same kind of natural-language type
// way.
var _ = Describe("Reconciler", func() {
	// NOTE(dperny): in these test descriptions, "resources" here is a
	// catch-all term for any service, network, config, secret, or other swarm
	// resource.
	var (
		// r is the reconciler object. we're testing it directly, not the
		// Reconciler interface, which is more for the benefit of external
		// users
		r *reconciler

		f *fakeReconcilerClient

		// we'll need a fake ObjectChangeNotifier, so we can construct the
		// Reconciler
		notifier *fakeObjectChangeNotifier

		stackFixture *interfaces.SwarmStack
	)

	BeforeEach(func() {
		// first things first, create a fakeReconcilerClient
		f = newFakeReconcilerClient()

		stackFixture = &interfaces.SwarmStack{
			ID: stackID,
			Spec: interfaces.SwarmStackSpec{
				Annotations: swarm.Annotations{
					Name:   stackName,
					Labels: map[string]string{},
				},
				Services: []swarm.ServiceSpec{},
				Networks: make(map[string]dockertypes.NetworkCreate),
				Secrets:  []swarm.SecretSpec{},
				Configs:  []swarm.ConfigSpec{},
			},
		}

		// first things first: create the notifier we'll be using. After test
		// code has executed, the caller can use this to verify the right
		// objects were called
		notifier = &fakeObjectChangeNotifier{}
	})

	JustBeforeEach(func() {
		// TODO(dperny): in this initial revision of tests, i'm building the
		// reconciler by hand, because i don't yet have a mock client to use
		r = newReconciler(notifier, f)
	})

	Describe("NewReconciler", func() {
		// This is, like, hello-world level stuff. Not actually a real test,
		// just getting everything bootstrapped.
		It("should return a new Reconciler object", func() {
			Expect(r).ToNot(BeNil())
		})
	})

	// TODO(dperny): specs marked "PIt" are "Pending" and do not execute.

	Describe("Reconciling a stack", func() {
		var (
			err error
		)
		BeforeEach(func() {
			// initialize the fixture services
			stackFixture.Spec.Services = append(stackFixture.Spec.Services,
				swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "service1-name",
						Labels: map[string]string{
							StackLabel: stackFixture.ID,
						},
					},
				},
				swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "service2-name",
						Labels: map[string]string{
							StackLabel: stackFixture.ID,
						},
					},
				},
			)

			// finally, put the stack in the fakeReconcilerClient
			f.stacksByName[stackFixture.Spec.Annotations.Name] = stackFixture.ID
			f.stacks[stackFixture.ID] = stackFixture
		})
		JustBeforeEach(func() {
			// this test handles all ReconcileStack cases, so its pretty
			// obvious that ReconcileStack is gonna be called for each of them
			err = r.Reconcile(StackEventType, stackID)
		})

		When("a new stack is created", func() {
			It("should create all of the objects defined within", func() {
				Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
			})
			It("should return no error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			When("resource creation fails", func() {
				BeforeEach(func() {
					// add the label "makemefail" to a service spec, which will
					// cause the fake to return an error
					stackFixture.Spec.Services[0].Annotations.Labels["makemefail"] = ""
				})
				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("a stack does not exit to be retrieved by the client", func() {
			BeforeEach(func() {
				// Actually no instead remove the stack
				delete(f.stacksByName, stackFixture.Spec.Annotations.Name)
				delete(f.stacks, stackFixture.ID)
			})
			It("should return no error", func() {
				// this is because if we returned an error when the stack was
				// not found, then we would try again to reconcile the stack,
				// and it would never succeed, totally locking up the
				// reconciler. the only way this can can have occurred is if
				// the stack was immediately deleted before we had a chance to
				// reconcile.
				Expect(err).ToNot(HaveOccurred())
			})
		})

		When("a stack cannot be retrieved for other reasons", func() {
			BeforeEach(func() {
				stackFixture.Spec.Annotations.Labels["makemefail"] = "yeet"
			})
			It("should return an error", func() {
				Expect(err).To(HaveOccurred())
			})
		})

		When("a service cannot be retrieved for some reason", func() {
			BeforeEach(func() {
				// create the service
				resp, _ := f.CreateService(stackFixture.Spec.Services[0], "", false)
				// get the service from the fakeReconcilerClient directly, so
				// we get the pointer
				service := f.services[resp.ID]
				service.Spec.Annotations.Labels["makemefail"] = ""
			})
		})

		When("a service for a stack already exist", func() {
			var (
				// serviceID is the ID of the service that will already exist
				serviceID string
			)

			BeforeEach(func() {
				resp, _ := f.CreateService(stackFixture.Spec.Services[0], "", false)
				serviceID = resp.ID
			})

			It("should notify the ObjectChangeNotifier that the service resources should be reconciled", func() {
				Expect(notifier.objects).To(ConsistOf(obj{"service", serviceID}))
			})

			It("should still create all of the other service", func() {
				Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
			})
		})

	})

	Describe("deleting a stack", func() {
		var (
			err error
		)
		BeforeEach(func() {
			specs := []swarm.ServiceSpec{
				{
					Annotations: swarm.Annotations{
						Name:   "service1",
						Labels: map[string]string{StackLabel: stackID},
					},
				},
				{
					Annotations: swarm.Annotations{
						Name:   "service2",
						Labels: map[string]string{StackLabel: "notthisone"},
					},
				}, {
					Annotations: swarm.Annotations{
						Name:   "service3",
						Labels: map[string]string{StackLabel: stackID},
					},
				}, {
					Annotations: swarm.Annotations{
						Name: "service4",
					},
				},
			}
			// Create some services belonging to a stack

			for _, spec := range specs {
				_, err := f.CreateService(spec, "", false)
				Expect(err).ToNot(HaveOccurred())
			}
		})
		JustBeforeEach(func() {
			err = r.Delete(StackEventType, stackID)
		})
		It("should return no error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("should notify the that all of the resources belonging to that stack should reconciled", func() {
			Expect(notifier.objects).To(ConsistOf(
				obj{"service", f.servicesByName["service1"]},
				obj{"service", f.servicesByName["service3"]},
			))
		})
	})

	Describe("Reconciling resources", func() {
		When("a resource is updated", func() {
			var (
				// kind and ID of the resource we'll be updating
				kind, id string
				err      error
			)
			JustBeforeEach(func() {
				err = r.Reconcile(kind, id)
			})

			When("the resource does not belong to a stack", func() {
				It("should return no error", func() {
					Expect(err).ToNot(HaveOccurred())
				})
			})
			PWhen("the resource belongs to a stack", func() {
				When("the resource does not match the stack definition", func() {
					PIt("should update the resource's spec")
					PIt("should return no error if successful")
					PIt("should return an error if unsuccessful")
				})
				When("the resource does match the stack's definition", func() {
					It("should return no error", func() {
						Expect(err).To(BeNil())
					})
					It("should perform no updates", func() {
						// covered by gomock
					})
				})
				When("the stack the resource belongs to does not exist", func() {
					PIt("should delete the resource", func() {})
				})
			})
		})

		When("a resource is deleted", func() {
			PIt("should notify the ObjectChangeNotifier that the stack should be reconciled", func() {
			})
		})
	})
})
