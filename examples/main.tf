terraform {
  required_providers {
    iterative = {
      versions = ["0.1"]
      source = "github.com/iterative/iterative"
    }
  }
}

provider "iterative" {}

resource "iterative_machine" "machine" {
  region = "us-west-1"
  aws_security_group = "default"
  key_public = "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAACAQCcztk+/ibMWJH7LTjcw5KhlDKW1y/gJVB3ivk3D0YMa84ylL5pc8/3zd4PkAvX4eJX0Yi/dE9r4dY+8+ws/1pIuAO9y9Fu3ev+Cj8CKbFbgxaLDlaWHCV/y295rhmgfArtTB4KCbQ2EihXgodzddA9FGiyPoeYWilUUNDsi9wBTsouGahaFpiDVeAkexkUgGtGUWTW7OcTgvagGmKoNogWEpo9VBU3gGEoWj/I1TecQmOs09NFMyj1DdtRfsKYhQUfYz1W38ht0zCuPHKOnVGLDK4vd3nI2KzKJu0/CcFbjlJNqPrHHooIGJtmQRQIsSyI6hCUPK3ZCI7o+viaGqP+Awbo7XSKyexqd81bhgha98jqy6304jOG5qSUewgeK7VNq2FEXQ0D7ox0Yci/TgM7w+XVpjOf6XEUjkUyoLoL1xkxcINdZozWzeXK/dykvfXo+nwALT4UhjMx7fk46e2lRyExBuD4L0ah8rDT1ZUORsDkEVvmCx/tJqO1drrUPLT846Cb0E6oebcYCUCN9r8qo2BeipG44VkX0jL9BLB2IZeP5BpXFT+bo3zjXqEtX3l/5iJ42jOJodmw70gaf/7c2NWumydR2STuDQLUSvJC2Xtka5M/CHfNX3ShssrJaR/oKacU8F5DaPqTH9RSJ3oSs8Kr247E20i1BzDTwrWicQ== g.ortega.david@gmail.com"
}
/* 
resource "iterative_machine" "machine2" {
  region = "us-west-1"
  aws_security_group = "default"
} */


# resource "iterative_machine" "machine3" {
#   region = "us-west-1"
# }

# resource "iterative_machine" "machine4" {
#   region = "us-west-2"
#   instance_hdd_size = 110
# }

/* resource "iterative_machine" "machine5" {
  region = "us-west-2"
} */

