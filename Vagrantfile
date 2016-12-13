Vagrant.configure("2") do |config|
  # TODO: Fix this when tinycorelinux.net isn't down
  # config.vm.provider "docker" do |d|
  #   d.image = "ubuntu/16.04"
  #   d.create_args = ["--privileged", "--cap-add=ALL"]
  # end

  config.vm.box = "bento/ubuntu-16.04"
  config.vm.provider "virtualbox" do |vb|
    disk_file = './tmp/disk.vdi'
    vb.customize ['createhd', '--filename', disk_file, '--size', 500 * 1024]
    vb.customize ['storageattach', :id, '--storagectl', 'SATA Controller', '--port', 1, '--device', 0, '--type', 'hdd', '--medium', disk_file]
    vb.customize ['modifyvm', :id, '--cableconnected1', 'on']
  end

  # Create a forwarded port mapping which allows access to a specific port
  # within the machine from a port on the host machine. In the example below,
  # accessing "localhost:8080" will access port 80 on the guest machine.
  # config.vm.network "forwarded_port", guest: 80, host: 8080

  # Enable provisioning with a shell script. Additional provisioners such as
  # Puppet, Chef, Ansible, Salt, and Docker are also available. Please see the
  # documentation for more information about their specific syntax and use.
  # config.vm.provision "shell", inline: <<-SHELL
  #   apt-get update
  #   apt-get install -y apache2
  # SHELL
end
