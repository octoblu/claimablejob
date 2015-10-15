package claimablejob_test

import (
	"strconv"
	"time"

	"github.com/garyburd/redigo/redis"
	"github.com/octoblu/claimablejob"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type DumbJob struct {}
func (dumbJob *DumbJob) GetKey() string {
	return "I am dumb"
}

var _ = Describe("ClaimableRedisJob", func() {
	var redisConn redis.Conn
	var sut claimablejob.ClaimableJob

	BeforeEach(func() {
		var err error

		redisConn, err = redis.Dial("tcp", ":6379")
		Expect(err).To(BeNil())
	})

	AfterEach(func(){
		redisConn.Close()
	})

	Describe("New", func() {
		Context("called with a key", func(){
			BeforeEach(func(){
				sut = claimablejob.New("old-map")
			})

			It("should set the key", func(){
				Expect(sut.GetKey()).To(Equal("old-map"))
			})
		})
	})

	Describe("NewFromJob", func() {
		Context("called with a job", func(){
			BeforeEach(func(){
				job := new(DumbJob)
				sut = claimablejob.NewFromJob(job)
			})

			It("should set the key", func(){
				Expect(sut.GetKey()).To(Equal("I am dumb"))
			})
		})
	})

	Describe("Claim", func(){
		Context("When the job is unset", func(){
			BeforeEach(func(){
				sut = claimablejob.New("faulty")
				_,err := redisConn.Do("DEL", "[namespace]-faulty")
				Expect(err).To(BeNil())
			})

			It("should return true", func(){
				Expect(sut.Claim()).To(BeTrue())
			})
		})

		Context("When the job has already run this second", func(){
			BeforeEach(func(){
				sut = claimablejob.New("faulty")
				then := int64(time.Now().Unix() + 1)
				_,err := redisConn.Do("SET", "[namespace]-faulty", then)
				Expect(err).To(BeNil())
			})

			It("should return false", func(){
				Expect(sut.Claim()).To(BeFalse())
			})
		})

		Context("When the job with a different name ran this second", func(){
			BeforeEach(func(){
				sut = claimablejob.New("smokey")
				_,err := redisConn.Do("DEL", "[namespace]-smokey")
				Expect(err).To(BeNil())
			})

			It("should return true", func(){
				Expect(sut.Claim()).To(BeTrue())
			})
		})

		Context("When the job ran in the previous second", func(){
			var gotClaim bool

			BeforeEach(func(){
				sut = claimablejob.New("faulty")
				now := int64(time.Now().Unix())
				_,err := redisConn.Do("SET", "[namespace]-faulty", now)
				Expect(err).To(BeNil())

				gotClaim,err = sut.Claim()
			})

			It("should return false", func(){
				Expect(gotClaim).To(BeTrue())
			})

			It("should advance the time", func(){
				result,err := redisConn.Do("GET", "[namespace]-faulty")
				Expect(err).To(BeNil())

				nextTickStr := string(result.([]uint8))
			  nextTick,err := strconv.ParseInt(nextTickStr, 10, 64)
				Expect(err).To(BeNil())

				then := int64(time.Now().Unix()) + 1
				Expect(nextTick).To(Equal(then))
			})
		})
	})
})
