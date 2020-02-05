resource "null_resource" "minion1" {
    triggers = {
        hostname = "minion1"
    }

    provisioner "salt-highstate" {
        address = "http://192.168.50.10:8000"
        username = "test_user"
        password = "test_pwd"
        backend = "pam"

        minion_id = self.triggers.hostname
    }
}
