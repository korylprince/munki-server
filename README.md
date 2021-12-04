# About

munki-server is an all-in-one server to deploy [Munki](https://www.munki.org/munki/) with three main parts:

* HTTP file server for Munki clients
* Simple dynamic manifest generation
* WebDAV server for mounting Munki repository as a file share - for use with munkitools, [MunkiAdmin](https://github.com/hjuutilainen/munkiadmin), etc

# Building

```bash
$ cd /path/to/build/directory
$ GOBIN="$(pwd)" go install "github.com/korylprince/munki-server@<tagged version>"
$ ./munki-server
```

# Configuring

munki-server is configured with environment variables:

Variable | Description | Default
-------- | ----------- | -------
WEBROOT | Path to root of Munki repository | Must be configured
MANIFESTROOT | Path relative from root to the manifests folder. Should have both leading and trailing slashes. For example, if WEBROOT is `/data` and the manifests folder is at `/data/repo/manifests`, MANIFESTROOT should be `/repo/manifests/` | Must be configured
ASSIGNMENTSPATH | Path to assignments configuration file (see below) | Must be configured
WEBDAVPREFIX | The path to access the WebDAV server. For example, if munki-server is hosted at `https://munki.example.com` and WEBDAVPREFIX is set to `/edit/`, the WebDAV share can be mounted at `https://munki.example.com/edit/` | /edit/
USERNAME | Username for WebDAV server | webdav
PASSWORD | Password for WebDAV server. If using the prebuilt Docker container, you can also specify PASSWORD_FILE for use with Docker secrets | Must be configured
PROXYHEADERS | Set to `true` if you want the server to rewrite IP addresses with X-Forwarded-For, etc headers | false
LISTENADDR | The host:port address you want the server to listen on | :80

# Dynamic Manifests

munki-server supports dynamic manifests to allow easy configuration for specific devices without having to manually create a lot of static files. ASSIGNMENTSPATH should point to a YAML file with the following schema:

```yaml
default:
  catalogs:
    - catalog1
    - catalog2
    - ...
  manifests:
    - manifest1
    - manifest2
    - ...
devices:
  - name: Text description
    client_identifier: ClientIdentifier
    catalogs:
      - catalog3
      - catalog4
    manifests:
      - manifest3
      - manifest4
```

munki-server will generate a manifest using the specified default catalogs and included manifests and return it to the client. If the client's ClientIdentifier is also configured under devices, those catalogs and included manifests will be merged into the default and return the result to the client.

If a client doesn't have the ClientIdentifier set, munki will use the device' serial number, so you can easily configure catalogs and included manifests per device. This method is recommended over manually setting a ClientIdentifier on each device.

Catalogs and included manifests are added to the generated manifest in the order they are specified in the assignments configuration (with defaults always before specific device configuration), and Munki will always use the last included manifest with the highest precedence.

If a manifest is requested that matches the name of a file in the manifests folder, dynamic generation will be skipped and the file will be sent like a normal web server.

## Example

Let's say you have a set of common software you want installed on all devices, but John Smith needs a special app. You would create two manifests: site_common (with all of the common software), and special_software (includes just the special software). Next you'd use the following assignments configuration:

```yaml
default:
  catalogs:
    - catalog_with_common_software
  manifests:
    - site_common
devices:
  - name: John Smith's MacBook
    client_identifier: <serial number>
    catalogs:
      - catalog_with_special_software
    manifests:
      - special_software
```

Normal clients would receive a generated manifest with site_common as an included manifest, while John Smith's MacBook would receive a manifest with both site_common and special_software as included manifests.

# Deploying

munki-server is intended to be deployed behind a reverse proxy with TLS termination (e.g. traefik, nginx, etc). Don't forget to set PROXYHEADERS to true if doing so.

There's a prebuilt Docker container at `korylprince/munki-server:<tagged version>`.
