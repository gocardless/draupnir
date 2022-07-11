# -*- mode: ruby -*-
# vi: set ft=ruby :
# frozen_string_literal: true

Vagrant.configure("2") do |config|
#  config.vm.box = "ubuntu/bionic64"
  config.vm.box = "generic/ubuntu1804"
  config.vm.network "forwarded_port", guest: 8443, host: 9443
  config.vm.network "forwarded_port", guest: 8080, host: 9080

  config.vm.synced_folder ".", "/draupnir", type: "rsync", rsync__exclude: ".git/"

  config.vm.provider "qemu" do |qe|
    qe.arch = "x86_64"
    qe.machine = "q35"
    qe.cpu = "max"
    qe.memory = "8G"
    qe.net_device = "virtio-net-pci"
  end

  # create disk
  data_disk_image = "./tmp/data_disk.vdi"
  config.vm.provider "virtualbox" do |v|
    unless File.exist?(data_disk_image)
      v.customize ["createhd", "--filename", data_disk_image, "--size", 1024]
    end
    v.customize [
      "storageattach", :id,
      "--storagectl", "SCSI",
      "--port", 2,
      "--device", 0,
      "--type", "hdd",
      "--medium", data_disk_image
    ]
  end

  config.vm.provision "shell", path: "vagrant/provision.sh"
end
