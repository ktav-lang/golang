require 'yaml'

path = './.github/workflows/native-distribution.yml'
workflow = YAML.load_file(path)
triggers = workflow['on'] || workflow[true]
abort 'missing native workflow contract' unless triggers.fetch('workflow_call').fetch('inputs').fetch('source-ref').fetch('required')
job = workflow.fetch('jobs').fetch('build-native')
settings = job.fetch('strategy').fetch('matrix').fetch('settings')
expected = %w[x86_64-unknown-linux-gnu aarch64-unknown-linux-gnu x86_64-apple-darwin aarch64-apple-darwin x86_64-pc-windows-gnullvm aarch64-pc-windows-gnullvm]
gnu_path = './.github/workflows/windows-gnu-cabi-candidate.yml'
gnu_call = workflow.fetch('jobs').fetch('windows-gnu')
abort 'wrong GNU producer caller' unless gnu_call.fetch('uses') == gnu_path
abort 'GNU source is not forwarded' unless gnu_call.fetch('with').fetch('source-ref') == '${{ inputs.source-ref }}'
gnu = YAML.load_file(gnu_path)
gnu_triggers = gnu['on'] || gnu[true]
abort 'missing GNU source contract' unless gnu_triggers.fetch('workflow_call').fetch('inputs').fetch('source-ref').fetch('required')
abort 'duplicate GNU PR producer' if gnu_triggers.key?('pull_request')
gnu_jobs = gnu.fetch('jobs')
gnu_targets = gnu_jobs.fetch('cabi-gnullvm').fetch('strategy').fetch('matrix').fetch('include').map { |item| item.fetch('target') }
abort 'wrong GNU target matrix' unless gnu_targets.sort == %w[x86_64-pc-windows-gnullvm aarch64-pc-windows-gnullvm].sort
abort 'incomplete combined native target matrix' unless (settings.map { |item| item.fetch('target') } + gnu_targets).sort == expected.sort
consumers = gnu_jobs.fetch('go-consumers')
abort 'missing GNU producer dependency' unless consumers.fetch('needs') == 'cabi-gnullvm'
actual = consumers.fetch('strategy').fetch('matrix').fetch('include').map { |item| [item.fetch('os'), item.fetch('arch'), item.fetch('go-arch'), item.fetch('target')] }
required = [['windows-latest', 'x64', 'amd64', 'x86_64-pc-windows-gnullvm'], ['windows-11-arm', 'arm64', 'arm64', 'aarch64-pc-windows-gnullvm']]
abort 'incomplete genuine Go consumer matrix' unless actual.sort == required.sort
staging = gnu_jobs.fetch('stage-release')
abort 'missing complete GNU consumer gate' unless staging.fetch('needs').sort == %w[cabi-gnullvm go-consumers].sort
abort 'incomplete GNU distribution target matrix' unless staging.fetch('strategy').fetch('matrix').fetch('target').sort == gnu_targets.sort
steps = staging.fetch('steps')
abort 'missing GNU staging command' unless steps.any? { |item| item['run'].to_s.include?('python -I scripts/stage_go_gnu_release.py') }
outputs = steps.select { |item| item['uses'].to_s.start_with?('actions/upload-artifact@') }
abort 'wrong GNU distribution artifact role' unless outputs.length == 1 && outputs[0].fetch('with').fetch('name') == 'native-distribution-${{ matrix.target }}' && !outputs[0].key?('if')
stager = File.read('scripts/stage_go_gnu_release.py')
%w[ktav_cabi-windows-amd64.dll ktav_cabi-windows-arm64.dll .NOTICE.json .NOTICE.manifest.json check_embedded_notices.py source_ref target].each do |role|
  abort "missing GNU delivery role #{role}" unless stager.include?(role)
end
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
