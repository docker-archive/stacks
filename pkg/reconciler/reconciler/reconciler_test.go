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

	dockertypes "github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/swarm"

	"github.com/docker/stacks/pkg/interfaces"
)

const (
	stackID   = "someIDwhocares"
	stackName = "someNamewhocares"
)

// obj creates a new objTuple. this allows us to use short-form struct
// initialization without tripping a linter error
func obj(kind, id string) objTuple {
	return objTuple{kind: kind, id: id}
}

// objTuple is just a tuple of 2 strings for use in the fakeObjectChangeNotifier
type objTuple struct {
	kind, id string
}

// fakeObjectChangeNotifier implements the ObjectChangeNotifier interface and
// keeps track of which objects it was notified about and in what order.
type fakeObjectChangeNotifier struct {
	objects []objTuple
}

func (f *fakeObjectChangeNotifier) Notify(kind, id string) {
	f.objects = append(f.objects, obj(kind, id))
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

		stackFixture *interfaces.Stack
	)

	BeforeEach(func() {
		// first things first, create a fakeReconcilerClient
		f = newFakeReconcilerClient()

		stackFixture = &interfaces.Stack{
			ID: stackID,
			Spec: interfaces.StackSpec{
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

	BeforeEach(func() {
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
							interfaces.StackLabel: stackFixture.ID,
						},
					},
				},
				swarm.ServiceSpec{
					Annotations: swarm.Annotations{
						Name: "service2-name",
						Labels: map[string]string{
							interfaces.StackLabel: stackFixture.ID,
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
			err = r.Reconcile(interfaces.StackEventType, stackID)
		})

		When("a new stack is created", func() {
			It("should create all of the objects defined within", func() {
				Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
			})
			It("should return no error", func() {
				Expect(err).ToNot(HaveOccurred())
			})
			It("should add a mapping of the service IDs to the stack", func() {
				for id := range f.services {
					Expect(r.stackResources[id]).To(Equal(stackID))
				}
			})
			When("resource creation fails", func() {
				BeforeEach(func() {
					// add the label "makemefail" to a service spec, which will
					// cause the fake to return an error
					stackFixture.Spec.Services[0].Annotations.Labels["makemefail"] = "invalidarg"
				})
				It("should return an error", func() {
					Expect(err).To(HaveOccurred())
				})
			})
		})

		When("a stack does not exist to be retrieved by the client", func() {
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
				stackFixture.Spec.Annotations.Labels["makemefail"] = "unavailable"
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
				service.Spec.Annotations.Labels["makemefail"] = "unavailable"
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
				Expect(notifier.objects).To(ConsistOf(obj("service", serviceID)))
			})

			It("should still create all of the other service", func() {
				Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
			})
		})

		When("a service with the stack label exists, but do not belong to a stack", func() {
			// This case is when some service with the stack label has been
			// created, but is not actually part of the stack spec. This might
			// happen, for example, if the stack is updated to remove some
			// service, but the reconciler is stopped before it can delete that
			// service
			var (
				serviceID string
			)

			BeforeEach(func() {
				resp, _ := f.CreateService(
					swarm.ServiceSpec{
						Annotations: swarm.Annotations{
							Name: "doesnotbelong",
							Labels: map[string]string{
								interfaces.StackLabel: stackFixture.ID,
							},
						},
					},
					"", false,
				)
				serviceID = resp.ID
			})

			It("should notify the ObjectChangeNotifier of the service", func() {
				Expect(notifier.objects).To(ConsistOf(obj("service", serviceID)))
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
						Labels: map[string]string{interfaces.StackLabel: stackID},
					},
				},
				{
					Annotations: swarm.Annotations{
						Name:   "service2",
						Labels: map[string]string{interfaces.StackLabel: "notthisone"},
					},
				}, {
					Annotations: swarm.Annotations{
						Name:   "service3",
						Labels: map[string]string{interfaces.StackLabel: stackID},
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
			err = r.Reconcile(interfaces.StackEventType, stackID)
		})
		It("should return no error", func() {
			Expect(err).ToNot(HaveOccurred())
		})
		It("should notify the that all of the resources belonging to that stack should reconciled", func() {
			Expect(notifier.objects).To(ConsistOf(
				obj("service", f.servicesByName["service1"]),
				obj("service", f.servicesByName["service3"]),
			))
		})
	})

	Describe("Reconciling services", func() {
		When("a service is updated", func() {
			var (
				id  string
				err error
			)
			JustBeforeEach(func() {
				err = r.Reconcile(events.ServiceEventType, id)
			})

			When("the service does not belong to a stack", func() {
				BeforeEach(func() {
					// create a service with no StackLabel
					resp, createErr := f.CreateService(swarm.ServiceSpec{
						Annotations: swarm.Annotations{
							Name: "foo",
						},
					}, "", false)
					Expect(createErr).ToNot(HaveOccurred())
					id = resp.ID
				})
				It("should return no error", func() {
					Expect(err).ToNot(HaveOccurred())
				})
			})

			When("the service belongs to a stack", func() {
				var (
					spec swarm.ServiceSpec
				)

				BeforeEach(func() {
					spec = swarm.ServiceSpec{
						Annotations: swarm.Annotations{
							Name:   "foo",
							Labels: map[string]string{interfaces.StackLabel: stackID},
						},
					}
					resp, createErr := f.CreateService(spec, "", false)
					Expect(createErr).ToNot(HaveOccurred())
					id = resp.ID
				})

				When("the stack has been deleted", func() {
					It("should delete the service", func() {
						// There should be no services in the fake anymore.
						Expect(f).To(ConsistOfServices([]swarm.ServiceSpec{}))
					})
				})

				When("the service does not match the stack definition", func() {
					BeforeEach(func() {
						differentSpec := spec
						// we have to make a new map for labels, because
						// otherwise we'd mutate the old map which is linked to
						// the old spec
						differentSpec.Annotations.Labels = map[string]string{
							interfaces.StackLabel: stackID,
							"klaatu":              "barada nikto",
						}
						stackFixture.Spec.Services = append(stackFixture.Spec.Services, differentSpec)
						f.stacks[stackFixture.ID] = stackFixture
						f.stacksByName[stackFixture.Spec.Annotations.Name] = stackFixture.ID
					})
					It("should update the resource's spec", func() {
						Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
						Expect(f.services[id].Meta.Version.Index).To(Equal(uint64(2)))
					})
					It("should return no error if successful", func() {
						Expect(err).ToNot(HaveOccurred())
					})
				})

				When("the service does not have a matching spec in the stack", func() {
					BeforeEach(func() {
						f.stacks[stackFixture.ID] = stackFixture
						f.stacksByName[stackFixture.Spec.Annotations.Name] = stackFixture.ID
					})
					It("should delete the service", func() {
						Expect(f).To(ConsistOfServices([]swarm.ServiceSpec{}))
					})
					It("should not return an error", func() {
						Expect(err).ToNot(HaveOccurred())
					})
				})

				When("the service does match the stack's definition", func() {
					BeforeEach(func() {
						stackFixture.Spec.Services = append(stackFixture.Spec.Services, spec)
						f.stacks[stackFixture.ID] = stackFixture
						f.stacksByName[stackFixture.Spec.Annotations.Name] = stackFixture.ID
					})
					It("should return no error", func() {
						Expect(err).To(BeNil())
					})
					It("should perform no updates", func() {
						Expect(f).To(ConsistOfServices(stackFixture.Spec.Services))
						// don't bother nil-checking in expectations, we'll
						// just allow the panic if it happens
						Expect(f.services[id].Meta.Version.Index).To(Equal(uint64(1)))
					})
				})
			})
		})

		When("a service is deleted", func() {
			var (
				err error
			)
			JustBeforeEach(func() {
				err = r.Reconcile(events.ServiceEventType, "gone")
			})
			When("the service belonged to no stack", func() {
				It("should return no error", func() {
					Expect(err).ToNot(HaveOccurred())
				})
			})
			When("the service belonged to a stack", func() {
				BeforeEach(func() {
					// Instead of going through the whole caching and deleting
					// process, just go into the object and add this service to
					// the cache
					r.stackResources["gone"] = stackID
				})
				It("should notify the ObjectChangeNotifier that the stack should be reconciled", func() {
					Expect(notifier.objects).To(ConsistOf(obj("stack", stackID)))
				})
				It("should return no error", func() {
					Expect(err).ToNot(HaveOccurred())
				})
				It("should clean up the stackResources entry for the service", func() {
					Expect(r.stackResources).ToNot(HaveKey("gone"))
				})
			})
		})
	})
})
