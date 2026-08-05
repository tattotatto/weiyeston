import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { Button } from 'antd';
import { ReloadOutlined } from '@ant-design/icons';
import type { Editor } from '@tiptap/core';

interface MobilePreviewProps {
  editor: Editor | null;
}

const PHONE_WIDTH = 375;
const PHONE_HEIGHT = 812;

const phoneFrameStyle: React.CSSProperties = {
  width: PHONE_WIDTH + 24,
  height: PHONE_HEIGHT + 48,
  backgroundColor: '#1a1a1a',
  borderRadius: 36,
  padding: '24px 12px',
  position: 'relative',
  boxShadow: '0 4px 24px rgba(0,0,0,0.25)',
};

const screenStyle: React.CSSProperties = {
  width: PHONE_WIDTH,
  height: PHONE_HEIGHT,
  border: 'none',
  backgroundColor: '#ffffff',
  borderRadius: 4,
  display: 'block',
};

const notchStyle: React.CSSProperties = {
  position: 'absolute',
  top: 12,
  left: '50%',
  transform: 'translateX(-50%)',
  width: 120,
  height: 6,
  backgroundColor: '#444',
  borderRadius: 3,
};

export function MobilePreview({ editor }: MobilePreviewProps) {
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const [html, setHtml] = useState('');

  useEffect(() => {
    if (!editor) {
      setHtml('');
      return;
    }

    const updatePreview = () => {
      setHtml(editor.getHTML());
    };

    editor.on('update', updatePreview);
    updatePreview();

    return () => {
      editor.off('update', updatePreview);
    };
  }, [editor]);

  const srcDoc = useMemo(() => {
    return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0, maximum-scale=1.0, user-scalable=no">
  <style>
    *, *::before, *::after {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }
    html {
      font-size: 16px;
      -webkit-text-size-adjust: 100%;
    }
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
        'Hiragino Sans GB', 'Microsoft YaHei', sans-serif;
      padding: 16px;
      line-height: 1.6;
      color: #333;
      min-height: 100vh;
    }
    img {
      max-width: 100%;
      height: auto;
    }
    video {
      max-width: 100%;
    }
    a {
      color: #1677ff;
    }
    blockquote {
      border-left: 4px solid #1677ff;
      padding-left: 12px;
      margin: 8px 0;
      color: #666;
    }
  </style>
</head>
<body>${html || '<p style="color:#999;text-align:center;padding-top:40px;">空白文档</p>'}</body>
</html>`;
  }, [html]);

  const handleRefresh = useCallback(() => {
    if (editor) {
      setHtml(editor.getHTML());
    }
  }, [editor]);

  return (
    <div
      style={{
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        gap: 16,
        padding: 24,
      }}
      data-testid="mobile-preview"
    >
      <div style={phoneFrameStyle}>
        <div style={notchStyle} data-testid="phone-notch" />
        <iframe
          ref={iframeRef}
          srcDoc={srcDoc}
          title="手机预览"
          style={screenStyle}
          data-testid="preview-iframe"
        />
      </div>
      <Button
        icon={<ReloadOutlined />}
        onClick={handleRefresh}
        data-testid="refresh-preview-button"
      >
        刷新预览
      </Button>
    </div>
  );
}
