package store

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/docker/stacks/pkg/interfaces"
)

var _ = Describe("StackStore", func() {
	It("should conform to the interfaces.StackStore interfaec", func() {
		// This doesn't actually contain any useful assertions, it'll just fail
		// at build time. However, we have to include at least one use of the
		// variable s or the build will also fail.
		var s interfaces.StackStore
		// create a new StackStore from scratch, instead of through the
		// constructor, because we don't have a client
		s = &StackStore{}
		Expect(s).ToNot(BeNil())
	})
})
