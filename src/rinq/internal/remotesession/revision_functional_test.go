// +build !without_amqp,!without_functests

package remotesession_test

import (
	"context"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/rinq/rinq-go/src/rinq"
	"github.com/rinq/rinq-go/src/rinq/internal/testutil"
)

//
var _ = Describe("revision (functional)", func() {
	var (
		ctx            context.Context
		ns             string
		client, server rinq.Peer
		session        rinq.Session
		local, remote  rinq.Revision
	)

	BeforeEach(func() {
		ctx = context.Background()
		ns = testutil.NewNamespace()
		client = testutil.NewPeer()
		session = client.Session()
		server = testutil.NewPeer()

		testutil.Must(server.Listen(ns, func(ctx context.Context, req rinq.Request, res rinq.Response) {
			remote = req.Source
			res.Close()
		}))

		local, _ = session.CurrentRevision()
		testutil.Must(session.Call(ctx, ns, "", nil))
	})

	AfterEach(func() {
		testutil.TearDownNamespaces()

		client.Stop()
		server.Stop()

		<-client.Done()
		<-server.Done()
	})

	Describe("Ref", func() {
		It("returns the same ref as the local revision", func() {
			Expect(remote.Ref()).To(Equal(local.Ref()))
		})
	})

	Describe("Refresh", func() {
		It("returns a revision with the same ref as the lastest local revision", func() {
			local, err := local.Update(ctx, ns, rinq.Set("a", "1"))
			Expect(err).NotTo(HaveOccurred())

			remote, err := remote.Refresh(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(remote.Ref()).To(Equal(local.Ref()))
		})
	})

	Describe("Get", func() {
		It("returns a previously updated value from the cache", func() {
			// setup a handler that updates an attribute remotely
			testutil.Must(server.Listen(ns, func(ctx context.Context, req rinq.Request, res rinq.Response) {
				var err error
				remote, err = req.Source.Update(ctx, ns, rinq.Set("a", "1"))
				Expect(err).NotTo(HaveOccurred())
				res.Close()
			}))

			// invoke the handler
			_, err := session.Call(ctx, ns, "", nil)
			Expect(err).NotTo(HaveOccurred())

			local, err := session.CurrentRevision()
			Expect(err).NotTo(HaveOccurred())

			// update the attribute locally such that the server does not know
			// about the change
			local, err = local.Update(ctx, ns, rinq.Set("a", "2"))
			Expect(err).NotTo(HaveOccurred())

			// the remote revision from the command handler should still be able
			// too pull the first value from its cache
			attr, err := remote.Get(ctx, ns, "a")
			Expect(err).NotTo(HaveOccurred())
			Expect(attr.Value).To(Equal("1"))
		})
	})

	// 	//
	// 	// Describe("ID", func() {
	// 	// 	It("returns a valid peer ID", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		err := subject.ID().Validate()
	// 	// 		Expect(err).ShouldNot(HaveOccurred())
	// 	// 	})
	// 	// })
	// 	//
	// 	// Describe("Session", func() {
	// 	// 	It("returns a session that belongs to this peer", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		Expect(sess.ID().Peer).To(Equal(subject.ID()))
	// 	// 	})
	// 	//
	// 	// 	It("returns a session with a non-zero seq component", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		Expect(sess.ID().Seq).To(BeNumerically(">", 0))
	// 	// 	})
	// 	//
	// 	// 	It("returns a session even if the peer is stopped", func() {
	// 	// 		subject := testutil.NewPeer()
	// 	//
	// 	// 		subject.Stop()
	// 	// 		<-subject.Done()
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		Expect(sess).ToNot(BeNil())
	// 	//
	// 	// 		sess.Destroy()
	// 	// 	})
	// 	// })
	// 	//
	// 	// Describe("Listen", func() {
	// 	// 	It("accepts command requests for the specified namespace", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		nonce := rand.Int63()
	// 	// 		err := subject.Listen(ns, testutil.AlwaysReturn(nonce))
	// 	// 		Expect(err).Should(BeNil())
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		p, err := sess.Call(context.Background(), ns, "", nil)
	// 	// 		defer p.Close()
	// 	//
	// 	// 		Expect(err).ShouldNot(HaveOccurred())
	// 	// 		Expect(p.Value()).To(BeEquivalentTo(nonce))
	// 	// 	})
	// 	//
	// 	// 	It("does not accept command requests for other namespaces", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		err := subject.Listen(ns, testutil.AlwaysPanic())
	// 	// 		Expect(err).Should(BeNil())
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	// 	// 		defer cancel()
	// 	//
	// 	// 		_, err = sess.Call(ctx, testutil.NewNamespace(), "", nil)
	// 	// 		Expect(err).To(Equal(context.DeadlineExceeded))
	// 	// 	})
	// 	//
	// 	// 	It("changes the handler when invoked a second time", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	// 		testutil.Must(subject.Listen(ns, testutil.AlwaysPanic()))
	// 	//
	// 	// 		nonce := rand.Int63()
	// 	// 		err := subject.Listen(ns, testutil.AlwaysReturn(nonce))
	// 	// 		Expect(err).Should(BeNil())
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		p, err := sess.Call(context.Background(), ns, "", nil)
	// 	// 		defer p.Close()
	// 	//
	// 	// 		Expect(err).ShouldNot(HaveOccurred())
	// 	// 		Expect(p.Value()).To(BeEquivalentTo(nonce))
	// 	// 	})
	// 	//
	// 	// 	It("returns an error if the namespace is invalid", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		err := subject.Listen("_invalid", testutil.AlwaysPanic())
	// 	// 		Expect(err).Should(HaveOccurred())
	// 	// 	})
	// 	//
	// 	// 	It("returns an error if the peer is stopped", func() {
	// 	// 		subject := testutil.NewPeer()
	// 	//
	// 	// 		subject.Stop()
	// 	// 		<-subject.Done()
	// 	//
	// 	// 		err := subject.Listen(ns, testutil.AlwaysPanic())
	// 	// 		Expect(err).Should(HaveOccurred())
	// 	// 	})
	// 	// })
	// 	//
	// 	// Describe("Unlisten", func() {
	// 	// 	It("stops accepting command requests", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	// 		testutil.Must(subject.Listen(ns, testutil.AlwaysPanic()))
	// 	//
	// 	// 		err := subject.Unlisten(ns)
	// 	// 		Expect(err).Should(BeNil())
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	// 	// 		defer cancel()
	// 	//
	// 	// 		_, err = sess.Call(ctx, ns, "", nil)
	// 	// 		Expect(err).To(Equal(context.DeadlineExceeded))
	// 	// 	})
	// 	//
	// 	// 	It("can be invoked when not listening", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		err := subject.Unlisten("unused-namespace")
	// 	// 		Expect(err).ShouldNot(HaveOccurred())
	// 	// 	})
	// 	//
	// 	// 	It("returns an error if the namespace is invalid", func() {
	// 	// 		subject := testutil.SharedPeer()
	// 	//
	// 	// 		err := subject.Unlisten("_invalid")
	// 	// 		Expect(err).Should(HaveOccurred())
	// 	// 	})
	// 	//
	// 	// 	It("returns an error if the peer is stopped", func() {
	// 	// 		subject := testutil.NewPeer()
	// 	// 		testutil.Must(subject.Listen(ns, testutil.AlwaysPanic()))
	// 	//
	// 	// 		subject.Stop()
	// 	// 		<-subject.Done()
	// 	//
	// 	// 		err := subject.Unlisten(ns)
	// 	// 		Expect(err).Should(HaveOccurred())
	// 	// 	})
	// 	// })
	// 	//
	// 	// Describe("Stop", func() {
	// 	// 	Context("when running normally", func() {
	// 	// 		It("cancels pending calls", func() {
	// 	// 			server := testutil.SharedPeer()
	// 	// 			barrier := make(chan struct{})
	// 	// 			testutil.Must(server.Listen(ns, testutil.Barrier(barrier)))
	// 	//
	// 	// 			subject := testutil.NewPeer()
	// 	//
	// 	// 			go func() {
	// 	// 				<-barrier
	// 	// 				subject.Stop()
	// 	// 				<-barrier
	// 	// 			}()
	// 	//
	// 	// 			sess := subject.Session()
	// 	// 			defer sess.Destroy()
	// 	//
	// 	// 			_, err := sess.Call(context.Background(), ns, "", nil)
	// 	// 			Expect(err).To(Equal(context.Canceled))
	// 	// 		})
	// 	// 	})
	// 	//
	// 	// 	Context("when stopping gracefully", func() {
	// 	// 		It("cancels pending calls", func() {
	// 	// 			server := testutil.SharedPeer()
	// 	// 			barrier := make(chan struct{})
	// 	// 			testutil.Must(server.Listen(ns, testutil.Barrier(barrier)))
	// 	//
	// 	// 			subject := testutil.NewPeer()
	// 	//
	// 	// 			go func() {
	// 	// 				<-barrier
	// 	// 				subject.GracefulStop()
	// 	// 				subject.Stop()
	// 	// 				<-barrier
	// 	// 			}()
	// 	//
	// 	// 			sess := subject.Session()
	// 	// 			defer sess.Destroy()
	// 	//
	// 	// 			_, err := sess.Call(context.Background(), ns, "", nil)
	// 	// 			Expect(err).To(Equal(context.Canceled))
	// 	// 		})
	// 	// 	})
	// 	// })
	// 	//
	// 	// Describe("GracefulStop", func() {
	// 	// 	It("waits for pending calls", func() {
	// 	// 		server := testutil.SharedPeer()
	// 	// 		barrier := make(chan struct{})
	// 	// 		testutil.Must(server.Listen(ns, testutil.Barrier(barrier)))
	// 	//
	// 	// 		subject := testutil.NewPeer()
	// 	//
	// 	// 		go func() {
	// 	// 			<-barrier
	// 	// 			subject.GracefulStop()
	// 	// 			<-barrier
	// 	// 		}()
	// 	//
	// 	// 		sess := subject.Session()
	// 	// 		defer sess.Destroy()
	// 	//
	// 	// 		_, err := sess.Call(context.Background(), ns, "", nil)
	// 	// 		Expect(err).ShouldNot(HaveOccurred())
	// 	// 	})
	// 	// })
})
