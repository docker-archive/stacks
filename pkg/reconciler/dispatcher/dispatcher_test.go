package dispatcher

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"time"

	"github.com/docker/stacks/pkg/mocks"
	"github.com/golang/mock/gomock"

	"github.com/docker/docker/api/types/events"

	"github.com/docker/stacks/pkg/interfaces"
	"github.com/docker/stacks/pkg/reconciler/notifier"
)

type fakeRegisterFunc func(notifier.ObjectChangeNotifier)

func (f fakeRegisterFunc) Register(n notifier.ObjectChangeNotifier) {
	if f != nil {
		f(n)
	}
}

var _ = Describe("Dispatcher", func() {
	var (
		mockCtrl *gomock.Controller
		// NOTE(dperny): I choose to use mocks for this test as the stand-in
		// for the the Reconciler, because this is a good use-case for mocks. I
		// want to be sure that the dispatcher calls only the specified methods
		// in the specified order with the specified arguments, and return the
		// specified result. Unlike in the Reconciler package, where the
		// implementation is decoupled from the end-result, the Dispatcher is
		// all about calling methods at the right time in the right order.
		mockReconciler *mocks.MockReconciler

		reg fakeRegisterFunc
	)

	BeforeEach(func() {
		mockCtrl = gomock.NewController(GinkgoT())
		mockReconciler = mocks.NewMockReconciler(mockCtrl)
	})

	Describe("creating a new dispatcher", func() {
		var (
			registered     bool
			registeredWith notifier.ObjectChangeNotifier
		)

		BeforeEach(func() {
			reg = func(n notifier.ObjectChangeNotifier) {
				registered = true
				registeredWith = n
			}
		})

		It("should register with the provided notifier.Register", func() {
			d := newDispatcher(mockReconciler, reg)
			Expect(registered).To(BeTrue())
			Expect(registeredWith).To(Equal(d))
		})
	})

	Describe("handling events", func() {
		var (
			d *dispatcher
		)

		BeforeEach(func() {
			d = newDispatcher(mockReconciler, reg)
		})

		It("should de-duplicate events", func() {
			// we should only get 1 call to Reconcile, even though we have 11
			// events for the same stack
			numberOfDuplicates := 11
			// create an event channel, with a buffer, and fill the buffer up
			eventC := make(chan interface{}, numberOfDuplicates)
			actor := events.Actor{
				ID: "someID",
			}

			for i := 0; i < numberOfDuplicates; i++ {
				eventC <- events.Message{
					Type:   interfaces.StackEventType,
					Action: "update",
					Actor:  actor,
				}
			}

			// we're expecting only 1 call to Reconcile
			mockReconciler.EXPECT().Reconcile(
				gomock.Eq(interfaces.StackEventType),
				gomock.Eq("someID"),
			).Return(nil)

			// TODO(dperny): this launches a goroutine, and goroutines in tests
			// are super duper unreliable and flaky. There must be a better
			// design here, but I don't know what it is.
			time.AfterFunc(3*time.Second, func() {
				close(eventC)
			})

			// now run HandleEvents
			err := d.HandleEvents(eventC)

			Expect(err).ToNot(HaveOccurred())

			// the mock will error if we try to call it more than once
		})
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})
})
