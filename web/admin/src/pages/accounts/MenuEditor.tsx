import { useState, useEffect, useCallback } from 'react';
import {
  Card, Button, Space, message, Tag, Modal, Form, Input, Select, Popconfirm, Empty, Spin, Row, Col,
} from 'antd';
import {
  PlusOutlined, DeleteOutlined, EditOutlined, ArrowLeftOutlined,
  SendOutlined, SaveOutlined, MenuOutlined, AppstoreOutlined,
} from '@ant-design/icons';
import { useParams, useNavigate } from 'react-router-dom';
import {
  getMenu, saveMenuDraft, publishMenu, deleteMenuDraft,
  type MenuVO,
} from '@/api/menu';

const { Option } = Select;

interface MenuButton {
  type: string;
  name: string;
  key?: string;
  url?: string;
  media_id?: string;
  appid?: string;
  pagepath?: string;
  sub_button?: MenuButton[];
}

interface MenuData {
  button: MenuButton[];
}

const MAX_BUTTONS = 3;
const MAX_SUB_BUTTONS = 5;

function MenuEditor() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const accountId = Number(id);

  const [loading, setLoading] = useState(false);
  const [saving, setSaving] = useState(false);
  const [publishing, setPublishing] = useState(false);
  const [menu, setMenu] = useState<MenuVO | null>(null);
  const [menuData, setMenuData] = useState<MenuData>({ button: [] });
  const [buttonModalOpen, setButtonModalOpen] = useState(false);
  const [editingButton, setEditingButton] = useState<{ parentIndex?: number; subIndex?: number } | null>(null);
  const [form] = Form.useForm();

  const fetchMenu = useCallback(async () => {
    setLoading(true);
    try {
      const res = await getMenu(accountId);
      const menuVO = res.data.data;
      setMenu(menuVO);
      if (menuVO && menuVO.menu_json) {
        try {
          const parsed = typeof menuVO.menu_json === 'string'
            ? JSON.parse(menuVO.menu_json)
            : menuVO.menu_json;
          setMenuData(parsed as MenuData);
        } catch {
          setMenuData({ button: [] });
        }
      } else {
        setMenuData({ button: [] });
      }
    } catch {
      message.error('获取菜单失败');
    } finally {
      setLoading(false);
    }
  }, [accountId]);

  useEffect(() => {
    fetchMenu();
  }, [fetchMenu]);

  const handleAddButton = (parentIndex?: number) => {
    setEditingButton(parentIndex !== undefined ? { parentIndex } : null);
    form.resetFields();
    form.setFieldsValue({ type: 'click' });
    setButtonModalOpen(true);
  };

  const handleEditButton = (parentIndex: number, subIndex: number) => {
    setEditingButton({ parentIndex, subIndex });
    const btn = menuData.button[parentIndex].sub_button![subIndex];
    form.setFieldsValue({
      name: btn.name,
      type: btn.type,
      key: btn.key,
      url: btn.url,
    });
    setButtonModalOpen(true);
  };

  const handleEditTopButton = (index: number) => {
    setEditingButton({ parentIndex: index });
    const btn = menuData.button[index];
    form.setFieldsValue({
      name: btn.name,
      type: btn.type,
      key: btn.key,
      url: btn.url,
    });
    setButtonModalOpen(true);
  };

  const handleButtonSubmit = () => {
    form.validateFields().then((values) => {
      const button: MenuButton = {
        type: values.type,
        name: values.name,
      };
      if (values.type === 'click') {
        button.key = values.key;
      } else if (values.type === 'view') {
        button.url = values.url;
      } else if (values.type === 'miniprogram') {
        button.url = values.url;
        button.appid = values.appid;
        button.pagepath = values.pagepath;
      }

      const newData = { ...menuData };
      if (editingButton) {
        if (editingButton.subIndex !== undefined) {
          // Editing sub button
          newData.button[editingButton.parentIndex!].sub_button![editingButton.subIndex] = button;
        } else if (editingButton.parentIndex !== undefined) {
          const existingBtn = newData.button[editingButton.parentIndex];
          if (existingBtn.sub_button && existingBtn.sub_button.length > 0) {
            // Convert sub_button menu → editing the top button name/type while keeping subs
            button.sub_button = existingBtn.sub_button;
          }
          newData.button[editingButton.parentIndex] = button;
        }
      } else {
        // Adding new top-level button
        if (newData.button.length >= MAX_BUTTONS) {
          message.error(`一级菜单最多 ${MAX_BUTTONS} 个`);
          return;
        }
        newData.button.push(button);
      }
      setMenuData(newData);
      setButtonModalOpen(false);
    });
  };

  const handleDeleteTopButton = (index: number) => {
    const newData = { ...menuData };
    newData.button.splice(index, 1);
    setMenuData(newData);
  };

  const handleDeleteSubButton = (parentIndex: number, subIndex: number) => {
    const newData = { ...menuData };
    newData.button[parentIndex].sub_button!.splice(subIndex, 1);
    if (newData.button[parentIndex].sub_button!.length === 0) {
      delete newData.button[parentIndex].sub_button;
    }
    setMenuData(newData);
  };

  const handleAddSubButton = (parentIndex: number) => {
    const btn = menuData.button[parentIndex];
    if (!btn.sub_button) {
      btn.sub_button = [];
    }
    if (btn.sub_button.length >= MAX_SUB_BUTTONS) {
      message.error(`子菜单最多 ${MAX_SUB_BUTTONS} 个`);
      return;
    }
    setEditingButton({ parentIndex, subIndex: btn.sub_button.length });
    form.resetFields();
    form.setFieldsValue({ type: 'click' });
    setButtonModalOpen(true);
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await saveMenuDraft(accountId, { menu_json: menuData });
      message.success('草稿已保存');
      fetchMenu();
    } catch {
      message.error('保存失败');
    } finally {
      setSaving(false);
    }
  };

  const handlePublish = async () => {
    setPublishing(true);
    try {
      await publishMenu(accountId);
      message.success('菜单已发布');
      fetchMenu();
    } catch {
      message.error('发布失败');
    } finally {
      setPublishing(false);
    }
  };

  const handleDeleteDraft = async () => {
    try {
      await deleteMenuDraft(accountId);
      message.success('草稿已删除');
      setMenu(null);
      setMenuData({ button: [] });
    } catch {
      message.error('删除失败');
    }
  };

  const buttonTypeLabels: Record<string, string> = {
    click: '点击事件',
    view: '跳转URL',
    miniprogram: '小程序',
    scancode_push: '扫码',
    scancode_waitmsg: '扫码(等待)',
    pic_sysphoto: '拍照',
    pic_photo_or_album: '拍照/相册',
    pic_weixin: '相册',
    location_select: '位置',
  };

  const renderButton = (btn: MenuButton, index: number) => {
    const hasSub = btn.sub_button && btn.sub_button.length > 0;
    return (
      <Card
        key={index}
        size="small"
        style={{ marginBottom: 8, backgroundColor: '#fafafa' }}
        title={
          <Space>
            <AppstoreOutlined />
            <strong>{btn.name || '(未命名)'}</strong>
            <Tag color="blue">{buttonTypeLabels[btn.type] || btn.type}</Tag>
            {hasSub && <Tag color="purple">含 {btn.sub_button!.length} 个子菜单</Tag>}
          </Space>
        }
        extra={
          <Space>
            <Button size="small" icon={<EditOutlined />} onClick={() => handleEditTopButton(index)}>
              编辑
            </Button>
            {!hasSub && (
              <Button size="small" icon={<PlusOutlined />} onClick={() => handleAddSubButton(index)}>
                添加子菜单
              </Button>
            )}
            <Popconfirm title="确定删除此按钮？" onConfirm={() => handleDeleteTopButton(index)}>
              <Button size="small" danger icon={<DeleteOutlined />}>删除</Button>
            </Popconfirm>
          </Space>
        }
      >
        {hasSub && (
          <div style={{ paddingLeft: 16 }}>
            {btn.sub_button!.map((subBtn: MenuButton, subIdx: number) => (
              <Card
                key={subIdx}
                size="small"
                style={{ marginBottom: 4 }}
                type="inner"
                title={
                  <Space>
                    <MenuOutlined />
                    {subBtn.name || '(未命名)'}
                    <Tag color="green">{buttonTypeLabels[subBtn.type] || subBtn.type}</Tag>
                  </Space>
                }
                extra={
                  <Space>
                    <Button size="small" onClick={() => handleEditButton(index, subIdx)}>编辑</Button>
                    <Popconfirm title="确定删除？" onConfirm={() => handleDeleteSubButton(index, subIdx)}>
                      <Button size="small" danger>删除</Button>
                    </Popconfirm>
                  </Space>
                }
              />
            ))}
            {btn.sub_button!.length < MAX_SUB_BUTTONS && (
              <Button type="dashed" size="small" block icon={<PlusOutlined />} onClick={() => handleAddSubButton(index)} style={{ marginTop: 4 }}>
                添加子菜单
              </Button>
            )}
          </div>
        )}
      </Card>
    );
  };

  if (loading) {
    return <Spin size="large" style={{ display: 'block', margin: '100px auto' }} />;
  }

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <Space>
          <Button icon={<ArrowLeftOutlined />} onClick={() => navigate(`/admin/accounts/${accountId}`)}>
            返回
          </Button>
          <h2 style={{ margin: 0 }}>自定义菜单管理</h2>
          {menu && (
            <Tag color={menu.status === 1 ? 'green' : 'default'}>
              {menu.status_text}
            </Tag>
          )}
        </Space>
        <Space>
          <Button
            icon={<SaveOutlined />}
            onClick={handleSave}
            loading={saving}
          >
            保存草稿
          </Button>
          <Button
            type="primary"
            icon={<SendOutlined />}
            onClick={handlePublish}
            loading={publishing}
            disabled={menuData.button.length === 0}
          >
            发布到微信
          </Button>
          {menu && menu.status === 0 && (
            <Popconfirm title="确定删除草稿？" onConfirm={handleDeleteDraft}>
              <Button danger icon={<DeleteOutlined />}>删除草稿</Button>
            </Popconfirm>
          )}
        </Space>
      </div>

      {menuData.button.length === 0 ? (
        <Empty
          description="暂未配置菜单"
          style={{ marginTop: 80 }}
        >
          <Button type="primary" icon={<PlusOutlined />} onClick={() => handleAddButton()}>
            添加菜单按钮
          </Button>
        </Empty>
      ) : (
        <Row gutter={[16, 16]}>
          <Col span={24}>
            <div style={{ marginBottom: 12 }}>
              <strong>一级菜单（最多 {MAX_BUTTONS} 个）</strong>
              {menuData.button.length < MAX_BUTTONS && (
                <Button
                  type="dashed"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => handleAddButton()}
                  style={{ marginLeft: 12 }}
                >
                  添加按钮
                </Button>
              )}
            </div>
            {menuData.button.map((btn, idx) => renderButton(btn, idx))}
          </Col>
        </Row>
      )}

      <Modal
        title={editingButton ? '编辑按钮' : '添加按钮'}
        open={buttonModalOpen}
        onOk={handleButtonSubmit}
        onCancel={() => setButtonModalOpen(false)}
        okText="确定"
        cancelText="取消"
        width={500}
        destroyOnClose
      >
        <Form form={form} layout="vertical">
          <Form.Item
            name="name"
            label="按钮名称"
            rules={[
              { required: true, message: '请输入按钮名称' },
              { max: 16, message: '按钮名称不能超过16个字符' },
            ]}
          >
            <Input placeholder="菜单显示名称" maxLength={16} showCount />
          </Form.Item>

          <Form.Item name="type" label="按钮类型" rules={[{ required: true }]}>
            <Select>
              <Option value="click">点击推事件</Option>
              <Option value="view">跳转URL</Option>
              <Option value="miniprogram">跳转小程序</Option>
              <Option value="scancode_push">扫码推事件</Option>
              <Option value="scancode_waitmsg">扫码带提示</Option>
              <Option value="pic_sysphoto">系统拍照发图</Option>
              <Option value="pic_photo_or_album">拍照或相册发图</Option>
              <Option value="pic_weixin">微信相册发图</Option>
              <Option value="location_select">发送位置</Option>
            </Select>
          </Form.Item>

          <Form.Item noStyle shouldUpdate={(prev, cur) => prev.type !== cur.type}>
            {({ getFieldValue }) => {
              const type = getFieldValue('type');
              if (type === 'click') {
                return (
                  <Form.Item name="key" label="事件KEY" rules={[{ required: true, message: '请输入事件KEY' }]}>
                    <Input placeholder="如: MENU_ABOUT" maxLength={128} />
                  </Form.Item>
                );
              }
              if (type === 'view') {
                return (
                  <Form.Item name="url" label="跳转URL" rules={[{ required: true, message: '请输入URL' }, { type: 'url', message: '请输入有效的URL' }]}>
                    <Input placeholder="https://..." maxLength={1024} />
                  </Form.Item>
                );
              }
              if (type === 'miniprogram') {
                return (
                  <>
                    <Form.Item name="url" label="页面路径" rules={[{ required: true, message: '请输入页面路径' }]}>
                      <Input placeholder="pages/index/index" />
                    </Form.Item>
                    <Form.Item name="appid" label="小程序AppId" rules={[{ required: true, message: '请输入AppId' }]}>
                      <Input placeholder="wx..." />
                    </Form.Item>
                    <Form.Item name="pagepath" label="备用网页" rules={[{ required: true, message: '请输入备用网页URL' }]}>
                      <Input placeholder="https://..." />
                    </Form.Item>
                  </>
                );
              }
              return null;
            }}
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

export default MenuEditor;
