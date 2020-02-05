# terraform-provisioner-salt-highstate

`salt-highstate` is a Terraform 0.12 provisioner that waits for a minion to connect to the master and applies highstate then reports whether it completed successfully.

The intended use-case is to integrate Terraform and SaltStack:
1. Create minion keys on master using [terraform-provider-salt](https://github.com/finarfin/terraform-provider-salt)

2. Configure and initialize Salt Minion using [cloud-init](https://cloudinit.readthedocs.io/en/latest/topics/modules.html#salt-minion).

3. Add salt-highstate provisioner to your manifests.

4. Terraform will create the machines, execute the provisioner and will only complete when the highstate is successfully applied. 
 
terraform-provisioner-salt-highstate requires Go version 1.13 or greater.

## Usage

### Installation
1. Download the binary for your platform to [Terraform plugins path](https://www.terraform.io/docs/plugins/basics.html#installing-plugins).
    - Windows: `%AppData%\terraform.d\plugins`
    - Linux: `~/.terraform.d/plugins`

2. Configure CherryPy NetAPI module on SaltStack master. See [setup section of Saltstack documentation](https://docs.saltstack.com/en/latest/ref/netapi/all/salt.netapi.rest_cherrypy.html#a-rest-api-for-salt) for instructions.

3. Add the provisioner to one of your resources. [See the example manifest](example/main.tf)

### Configuration
| Name | Type | Required | Remarks |
|-|-|-|-|
| `address` | String | **Required** | URL to the CherryPy NetAPI endpoint (e.g.: https://saltmaster:8000) | `username`| String | **Required** | Username |
| `password`| String | **Required** | Password |
| `backend` | String | **Required** | External authentincation backend (eauth) (e.g.: pam) |
| `minion_id` | String | **Required** | Minion ID |
| `timeout_minutes` | Int | *Optional* | No of minutes to wait for minion to become available (Default: 30 mins) |
| `interval_secs` | Int | *Optional* | Interval in seconds to poll minion status from master (Default: 10 secs) |

## Example
```terraform
resource "null_resource" "test" {
    triggers = {
        hostname = "minion${count.index+1}"
    }

    provisioner "salt-highstate" {
        address = "https://salt-master:8000"
        username = "test"
        password = "test"
        backend = "pam"
        minion_id = self.triggers.hostname
    }

    count = 2
}
```
