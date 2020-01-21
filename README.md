# terraform-provisioner-salt-highstate

Experimental Terraform provisioner that waits for a minion to connect to the master and applies highstate then reports whether it completed successfully.

terraform-provisioner-salt-highstate requires Go version 1.13 or greater.

## Usage

```terraform
resource "null_resource" "test" {
    triggers = {
        minion_id = "minion${count.index+1}"
    }

    provisioner "salt-highstate" {
        address = "https://salt-master:8000"
        username = "test"
        password = "test"
        backend = "pam"
        minion_id = self.triggers.minion_id
    }

    count = 2
}
```