export PKR_VAR_image_name=iterative-cml-$(cat /proc/sys/kernel/random/uuid)
export PKR_VAR_test=true
packer build -force .
