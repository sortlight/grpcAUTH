# My gRPC Authentication 

This repo started as a simple weekend project to wrap my head around gRPC authentication in Go which is to secure a service with mutual TLS (mTLS) and JWT tokens. Then I added roles for access control, some cool streaming features and metrics to monitor it all.  Before I knew it, this little demo evolved into a powerhouse showcase for microservices skills. It's got everything from RBAC to bidirectional streaming, Prometheus observability and even a handy CLI tool to tie it all together. Enjoy the ride!

Let me take you through the story...

## The Spark

Security isn't just about locking the door. it's about who gets the key. That's where RBAC came in. I added roles like "admin" and "user" to control access to different RPCs.

I thought "gRPC shines with streaming!" So I sprinkled in server streaming, client streaming and bidirectional RPCs to handle things like real time chats or batch processing. Observability? Non-negotiable for production. I hooked up Prometheus metrics to track requests, errors and latencies because who wants a blackbox service?

It also include a complementary CLI tool using Cobra. It's not just a toy, it lets you interact with the gRPC server from the command line, perfect for scripting or debugging. This combo screams "I can build APIs *and* tools!" Yes that is where I'm going with this.

If you're like me who is curious about secure distributed systems. this project's for you. Fork it and play around with it.

## The Features I Added Along the Way
- **Mutual TLS (mTLS)**: The foundation. Encrypts everything and verifies identities with certs. No more plain-text snooping!
- **JWT Authentication with RBAC**: Tokens carry roles (e.g., "admin" can do everything, "user" only basics). An interceptor checks permissions per RPC. Inspired by real-world setups like those at Netflix or Uber.
- **Streaming RPCs**: 
  - Server-streaming: Get a barrage of greetings.
  - Client-streaming: Send a stream of names, get one summary response.
  - Bidirectional: Chat-like exchange—send and receive in real-time.
- **Prometheus Observability**: Metrics for request counts, durations, and error rates. Exposed at `/metrics` for scraping by Prometheus. I used interceptors to keep it clean and automatic.
- **CLI Tool**: A Cobra-based command-line interface to call the server. Run `grpc-cli greet --name World` or stream chats. Shows off Go's CLI prowess alongside gRPC.



# Author 

Made by [Sortlight](https://github.com/sortlight/) 