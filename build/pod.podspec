Pod::Spec.new do |spec|
  spec.name         = 'Gfbc'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/fairblock/go-fairblock'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Fairblock Client'
  spec.source       = { :git => 'https://github.com/fairblock/go-fairblock.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gfbc.framework'

	spec.prepare_command = <<-CMD
    curl https://gfbcstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gfbc.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
