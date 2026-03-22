module.exports = {
  '*.go': () => 'golangci-lint run ./...',
  'proto/**/*.proto': () => 'buf lint proto',
  'web/**/*.{ts,tsx}': [
    'prettier --write',
  ],
  'web/**/*.{json,css,md}': 'prettier --write',
};
