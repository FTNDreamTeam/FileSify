<div align="center">

# ☁️ FileSify 

**A highly scalable, fault-tolerant file storage and sharing platform built with Go & Service-Oriented Architecture.**

[![Go Version](https://img.shields.io/badge/Go-1.27+-00ADD8?style=for-the-badge&logo=go)](https://golang.org)
[![gRPC](https://img.shields.io/badge/gRPC-Protocol-244c5a?style=for-the-badge&logo=grpc)](https://grpc.io)
[![Docker](https://img.shields.io/badge/Docker-Enabled-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com)
[![MongoDB](https://img.shields.io/badge/MongoDB-4EA94B?style=for-the-badge&logo=mongodb&logoColor=white)](https://www.mongodb.com)
[![Silo](https://img.shields.io/badge/Silo_S3_Storage-C7202C?style=for-the-badge&logo=minio)](https://silo.pgsty.com/)
[![MailHog](https://img.shields.io/badge/MailHog-SMTP_Testing-FF6C37?style=for-the-badge&logo=maildotru&logoColor=white)](https://github.com/mailhog/MailHog)


### Frontend
[![HTML5](https://img.shields.io/badge/HTML5-E34F26?style=for-the-badge&logo=html5&logoColor=white)]()
[![CSS3](https://img.shields.io/badge/CSS3-1572B6?style=for-the-badge&logo=css3&logoColor=white)]()
[![JavaScript](https://img.shields.io/badge/JavaScript-F7DF1E?style=for-the-badge&logo=javascript&logoColor=black)]()
[![Bootstrap](https://img.shields.io/badge/Bootstrap-563D7C?style=for-the-badge&logo=bootstrap&logoColor=white)]()
</div>

---

## 📖 Overview

**FileSify** is a Dropbox-inspired distributed system designed to demonstrate modern microservice architecture. It utilizes **gRPC** for blazing-fast inter-service communication and a **gRPC-Gateway** to expose a RESTful API to a unified web frontend. 

The system guarantees high availability through **fault-tolerant retry loops**, isolated database states, and strict **Docker network segregation**.

---

## 📸 Preview

<div align="center">
  <img src="./assets/filesify-preview.png" alt="FileSify Landing Page" width="100%" style="border-radius: 10px; box-shadow: 0 4px 8px rgba(0,0,0,0.1);">
</div>

---

## 🏗️ System Architecture

<div align="center">
  <img src="./assets/filesify-diagram.png" alt="FileSify Landing Page" width="60%" style="border-radius: 10px; box-shadow: 0 4px 8px rgba(0,0,0,0.1);">
</div>

### 🧩 Microservices Breakdown

| Service | Responsibility | Tech Stack | Port |
| :--- | :--- | :--- | :--- |
| 🛡️ **API Gateway** | REST to gRPC translation, serves static frontend files. | Go, gRPC-Gateway | `8080` |
| 👤 **User Service** | Auth, JWT generation, 2FA, user profiles. | Go, MongoDB | `8082` |
| 🗄️ **Storage Service** | File & folder metadata, infinite nested directory logic. | Go, S3 API (Silo) | `8083` |
| ✉️ **Mail Service** | Dispatching OTPs and Magic Links. | Go, SMTP | `8085` |

### 🌐 Network Topology
To ensure zero-trust security, the services are segmented into three distinct Docker networks:
*   🟢 **`user-network`**: Isolates User DB, User Service, and Mail Service.
*   🟡 **`storage-network`**: Isolates the Storage Service and the MinIO/Silo object store.
*   🔵 **`mail-network`**: Isolates the Mail Service and the MailHog SMTP server.

---

## 🚀 Getting Started

### 📋 Prerequisites
Ensure you have the following installed on your machine:
*   [Docker & Docker Compose](https://docs.docker.com/get-docker/)
*   [Go (1.27+)](https://go.dev/dl/)
*   [Protocol Buffers Compiler](https://grpc.io/docs/protoc-installation/)

### ⚙️ Environment Setup
Create the necessary `.env` files in your `backend` directory before launching the stack.

<details>
<summary><b>Click to view backend/.env configuration</b></summary>

```env
# --- Object Storage (Silo/MinIO) ---
MINIO_ROOT_USER=silo-admin
MINIO_ROOT_PASSWORD=change-me
SILO_ENDPOINT=silo:9000
SILO_CONSOLE_PORT=9001
STORAGE_PORT=8083

# --- Gateway & User Service ---
GATEWAY_PORT=8080
USER_SERVICE_PORT=8082
USER_DB_USERNAME=user_admin
USER_DB_PASS=user_pass