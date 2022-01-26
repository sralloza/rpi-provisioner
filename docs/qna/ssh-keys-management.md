# SSH Keys management

I have some PCs, each with a differnet SSH key. Naturally I would want to be able to SSH into the Raspberry from any of my PCs. But, what happens if I change one key? Do I have to manually add the key to each Raspberry I have?

With [rpi-provisioner](./), no. You just have to change your public SSH in the file. You just need to write your public SSH keys into a json file and upload it to an s3 bucket.

Note: if you don't want to use AWS S3 to store your keys file, use the `--keys-path` command to specify the path to the file where you store your public keys.

Example:

```json
{
  "key-id-1": "public-ssh-key-1",
  "key-id-2": "public-ssh-key-2"
}
```

Then you set your AWS env vars (`$AWS_ACCESS_KEY_ID` and `$AWS_SECRET_ACCESS_KEY`). If you don't have them, a simple google search will tell you how to generate them. You will need to tell the script where your file containing the public SSH keys is in AWS. You do it with the `--s3-path` flag: `--s3-path=<REGION>/<BUCKET_NAME>/<FILE_NAME>`. If you don't use this convention the script will complain and raise an error.

We have covered how to store your public SSH keys. How can you update the SSH keys in your raspberry's authorized_keys? Simple, just use the [authorized-keys](#authorized-keys) command (or the [layer1](#layer1) command if you set up the raspberry for the first time).
