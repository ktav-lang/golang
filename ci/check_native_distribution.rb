require 'yaml'

path = './.github/workflows/native-distribution.yml'
workflow = YAML.load_file(path)
triggers = workflow['on'] || workflow[true]
abort 'missing native workflow contract' unless triggers.fetch('workflow_call').fetch('inputs').fetch('source-ref').fetch('required')
job = workflow.fetch('jobs').fetch('build-native')
settings = job.fetch('strategy').fetch('matrix').fetch('settings')
expected = %w[x86_64-unknown-linux-gnu aarch64-unknown-linux-gnu x86_64-apple-darwin aarch64-apple-darwin x86_64-pc-windows-msvc aarch64-pc-windows-msvc]
abort 'incomplete native target matrix' unless settings.map { |item| item.fetch('target') }.sort == expected.sort
abort 'incompatible Go macOS runner' unless settings.select { |item| item.fetch('target').include?('apple') }.all? { |item| item.fetch('os') == 'macos-15' }
caller = YAML.load_file('.github/workflows/native-distribution-check.yml').fetch('jobs').fetch('native-distribution')
abort 'wrong native CI caller' unless caller.fetch('uses') == path
abort 'native CI source is not published Go 0.8.1' unless caller.fetch('with').fetch('source-ref') == 'dd7a14cc1c88199b5515f5c891100dd72ef211fd'
release = YAML.load_file('.github/workflows/release.yml').fetch('jobs')
producer = release.fetch('build-native')
abort 'production native build differs from CI' unless producer.fetch('uses') == path
abort 'production source must follow caller commit' unless producer.fetch('with').fetch('source-ref') == '${{ github.sha }}'
abort 'missing native publication gate' unless Array(release.fetch('release').fetch('needs')).include?('build-native')
downloads = release.fetch('release').fetch('steps').select { |step| step['uses'].to_s.start_with?('actions/download-artifact@') }
selectors = downloads.map { |step| step.fetch('with').slice('name', 'pattern') }
abort 'unsafe release artifact download' unless selectors == [{'pattern' => 'native-distribution-*'}]
uploads = job.fetch('steps').select { |step| step['uses'].to_s.start_with?('actions/upload-artifact@') }
distribution = uploads.find { |step| step.fetch('with').fetch('name').start_with?('native-distribution-') }
abort 'missing native notice sidecars' unless distribution && distribution.fetch('with').fetch('path').include?('.NOTICE.json') && distribution.fetch('with').fetch('path').include?('.NOTICE.manifest.json')
steps = job.fetch('steps')
%w[Prepare\ locked\ native\ notice\ resource Verify\ embedded\ resource\ and\ write\ distribution\ sidecars Run\ Go\ tests\ on\ matching\ target\ architecture].each do |name|
  step = steps.find { |item| item['name'] == name }
  abort "missing mandatory step: #{name}" unless step && !step.key?('if') && !step['continue-on-error']
end
puts 'native distribution workflow and publication gates verified'
