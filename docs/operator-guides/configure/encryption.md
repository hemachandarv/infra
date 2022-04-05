# Encryption Keys

Sensitive data is always encrypted at rest in the db using a symmetric key. The symmetric key is stored in the database encrypted by a root key. By default this root key is generated by Infra and stored in a secret (default: `~/.infra/key`, or in Kubernetes, a as secret named `infra-x` with the key `/__root_key`). Encrpytion at rest can be configured using another key provider service such as KMS or Vault.

The process of retrieving the db key is to load the encrypted key from the database, request that the db key be decrypted by the root key, and at which point the db key is used to decrypt all the data. In the case of AWS KMS and Vault, the Infra app never sees the root key, and so these options are preferred over the default built-in `native` key provider.

### Root key configuration examples

Infra uses AWS KMS key service:

```yaml
server:
  config:
    keys:
      - kind: awskms
        endpoint: https://your.kms.aws.url.example.com
        region: us-east-1
        accessKeyId: kubernetes:awskms/accessKeyID
        secretAccessKey: kubernetes:awskms/secretAccessKey
        encryptionAlgorithm: AES_256
```

Infra uses Vault as a key service:

```yaml
server:
  config:
    keys:
      - kind: vault
        address: https://your.vault.url.example.com
        transitMount: /transit
        token: kubernetes:vault/token
        namespace: namespace
```

By default, Infra will manage keys internally. You can use a predefined 256-bit cryptographically random key by creating and mounting a secret to the server pod.

```yaml
server:
  volumes:
    - name: my-db-encryption-secret
      secret:
        secretName: my-db-encryption-secret
  volumeMounts:
    - name: my-encryption-secret
      mountPath: /var/run/secrets/my/db/encryption/secret

  config:
    dbEncryptionKey: /var/run/secret/my/db/encryption/secret
```

If an encryption key is not provided, one will be randomly generated during install time. It is the responsibility of the operator to back up this key.