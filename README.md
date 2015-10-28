The poor man's dynamic dns for custom domains.

#About

Pmdns is a daemon that periodically sets DNS records for the machine's external
IP address.  Typically pmdns would be run inside a LAN connected to the
internet using a dynamic IP address.

The method to obtain the IP address is configurable and can vary in complexity
and security.

- Scraping well-known websites.
- An interface IP address.

#Installation

Download a binary distribution

#Configuration

Pmdns is configured through a toml file at /etc/pmdns/config.toml

    [DetectionService]
    Type = "URL"
    URL.HRef = "http://ifconfig.me/ip"
    URL.ExtractPattern = "(\\.*)"

    [Provisioner.aws]
    Provider = "aws-route53"
    BotoProfile = "myprofile"

    [Provisioner.aws.Records.example]
    Record = "example.com"
