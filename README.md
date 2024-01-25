# Cucurbita

A reliable, low-latency, and anti-censorship virtual private network controller.

## Why a new version is needed

For me personally, I want to know how many users a product has, such as current online users, daily active users, and weekly active users.
For network administrators, they also need to be able to control and manage devices in the network.
It is a bit complicated to use C++ to complete these functions, but it is just right to use other upper-level languages. Therefore, I decided to use Go to re-develop it.

## How to use

Create data storage directory

```bash
mkdir cucurbita
```

Map port 8080 to port 80 inside the container, and map the data storage directory to the container.

```bash
docker run --rm -p 8080:80 -v cucurbita:/var/lib/cucurbita docker.io/lanthora/cucurbita:latest
```

Now you can access this web service. You can enter any password when logging in for the first time, and this password will be the password for subsequent logins.
