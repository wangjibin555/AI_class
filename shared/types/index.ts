/**
 * 共享类型定义
 */

// 文档类型
export enum DocumentType {
  DOCX = 'docx',
  PDF = 'pdf',
  MARKDOWN = 'md',
  TXT = 'txt',
}

// 任务状态
export enum TaskStatus {
  PENDING = 'pending',
  PROCESSING = 'processing',
  COMPLETED = 'completed',
  FAILED = 'failed',
}

// 用户角色
export enum UserRole {
  ADMIN = 'admin',
  USER = 'user',
  GUEST = 'guest',
}

// 文档上传请求
export interface UploadRequest {
  file: File;
  userId: string;
  options?: {
    template?: string;
    style?: string;
  };
}

// 任务信息
export interface Task {
  id: string;
  userId: string;
  documentId: string;
  status: TaskStatus;
  progress: number;
  createdAt: Date;
  updatedAt: Date;
  resultUrl?: string;
  error?: string;
}

// 文档信息
export interface Document {
  id: string;
  userId: string;
  filename: string;
  type: DocumentType;
  size: number;
  url: string;
  createdAt: Date;
}

// PPT生成配置
export interface PPTConfig {
  template: string;
  style: {
    theme: string;
    fontSize: number;
    fontFamily: string;
  };
  layout: {
    titleSlide: boolean;
    tocSlide: boolean;
  };
}

// 内容块
export interface ContentChunk {
  type: 'title' | 'text' | 'list' | 'image' | 'table';
  content: string | string[];
  metadata?: Record<string, any>;
}

// 解析后的文档结构
export interface ParsedDocument {
  title: string;
  author?: string;
  sections: Section[];
}

export interface Section {
  title: string;
  level: number;
  content: ContentChunk[];
  subsections?: Section[];
}

// API响应
export interface ApiResponse<T = any> {
  success: boolean;
  data?: T;
  error?: {
    code: string;
    message: string;
  };
}

// 用户信息
export interface User {
  id: string;
  username: string;
  email: string;
  role: UserRole;
  createdAt: Date;
}

// JWT载荷
export interface JWTPayload {
  userId: string;
  username: string;
  role: UserRole;
  iat: number;
  exp: number;
}


