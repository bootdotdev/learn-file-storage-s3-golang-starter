# aws-s3
## Single Machine
Why is S3 being "serverless" a big deal?

In a "simple" web application architecture, your server is likely a single machine running in the cloud. That single machine probably runs:
1. An HTTP server that handles incoming requests.
2. A database running in the background that the HTTP server talks to.
3. A file system the server uses to directly read and write to larger files.

All on one machine.

### Whats wrong with this?
Well, not much. Honestly this is a perfectly valid way to build a web application, even in production. That said, there are some trade-offs:
* **Scaling**: If your app gets popular, you'll need to "scale" your single machine (add more resources like CPU/RAM/Disk space). A single computer can only become so powerful.
* **Availability**: If your server goes down, your app goes down. To be fair, you can mitigate this with load balancers and multiple servers.
* **Durability**: If your server crashes, or an intern `rm -rf`'s something, you're in trouble. You might have backups, but let's be honest, you probably don't.
* **Cost**: Running a server 24/7 means paying 24/7. It can be nice to only pay for what you use.
* **Maintenance**: You have to manage everything yourself. You'll be responsible for "ops" tasks like backups, monitoring, logging, version upgrades, etc.

## AWS
AWS is one of the "Big Three" cloud providers. The other two are Google Cloud and Microsoft Azure.

AWS is the oldest, largest, and most popular of the three.
