#!/bin/bash

# 生成包含正确IP地址的新证书
echo "生成包含当前IP 10.30.15.93 的新证书..."

# 创建配置文件
cat > cert_new.conf << EOL
[req]
default_bits = 4096
prompt = no
distinguished_name = req_distinguished_name
req_extensions = v3_req

[req_distinguished_name]
C=CN
ST=Beijing
L=Beijing
O=AI-Classroom
OU=Development
CN=localhost

[v3_req]
keyUsage = critical, digitalSignature, keyEncipherment, keyAgreement
extendedKeyUsage = critical, serverAuth, clientAuth
subjectAltName = @alt_names
basicConstraints = CA:FALSE

[alt_names]
DNS.1 = localhost
DNS.2 = 127.0.0.1
DNS.3 = 10.30.26.56
DNS.4 = 10.30.15.93
IP.1 = 127.0.0.1
IP.2 = 10.30.26.56
IP.3 = 10.30.15.93
EOL

# 备份旧证书
cp certs/cert.pem certs/cert.pem.backup
cp certs/key.pem certs/key.pem.backup

# 生成新私钥
openssl genrsa -out certs/key.pem 4096

# 生成新证书
openssl req -new -x509 -key certs/key.pem -out certs/cert.pem -days 365 -config cert_new.conf -extensions v3_req

# 清理配置文件
rm cert_new.conf

echo "✅ 新证书生成完成，包含当前IP地址"
echo "证书信息："
openssl x509 -in certs/cert.pem -text -noout | grep -A 3 "Validity"
openssl x509 -in certs/cert.pem -text -noout | grep -A 5 "Subject Alternative Name"

