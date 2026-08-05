import { useState } from 'react';
import { Form, Input, InputNumber, Select, DatePicker, Button, Space, message, Card, Switch, Divider } from 'antd';
import { PlusOutlined, MinusCircleOutlined, ArrowLeftOutlined } from '@ant-design/icons';
import { useNavigate } from 'react-router-dom';
import { createVote, type CreateVoteParams } from '@/api/vote';

const { TextArea } = Input;

function VoteCreate() {
  const navigate = useNavigate();
  const [form] = Form.useForm();
  const [loading, setLoading] = useState(false);

  const handleSubmit = async () => {
    setLoading(true);
    try {
      const values = await form.validateFields();
      const params: CreateVoteParams = {
        title: values.title,
        description: values.description || undefined,
        cover_url: values.cover_url || undefined,
        vote_type: values.vote_type || 1,
        max_choices: values.max_choices || 1,
        max_votes: values.max_votes || 1,
        start_time: values.time_range?.[0]?.toISOString(),
        end_time: values.time_range?.[1]?.toISOString(),
        status: values.status !== undefined ? (values.status ? 1 : 0) : 0,
        options: (values.options || []).map((opt: { content: string; image_url?: string }, idx: number) => ({
          content: opt.content,
          image_url: opt.image_url || undefined,
          sort_order: idx,
        })),
      };

      await createVote(params);
      message.success('投票已创建');
      navigate('/votes');
    } catch (err: any) {
      if (err.errorFields) return;
      message.error('创建失败');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div style={{ maxWidth: 700 }}>
      <Space style={{ marginBottom: 16 }}>
        <Button icon={<ArrowLeftOutlined />} onClick={() => navigate('/votes')}>返回</Button>
      </Space>

      <Card title="创建投票">
        <Form form={form} layout="vertical" initialValues={{ vote_type: 1, max_choices: 1, max_votes: 1, status: false }}>
          <Form.Item name="title" label="投票标题" rules={[{ required: true, message: '请输入投票标题' }]}>
            <Input placeholder="请输入投票标题" maxLength={200} />
          </Form.Item>

          <Form.Item name="description" label="投票说明">
            <TextArea rows={3} placeholder="投票规则说明" maxLength={500} />
          </Form.Item>

          <Form.Item name="cover_url" label="封面图 URL">
            <Input placeholder="可选封面图 URL" />
          </Form.Item>

          <Space size="large" wrap>
            <Form.Item name="vote_type" label="投票类型">
              <Select style={{ width: 120 }}>
                <Select.Option value={1}>单选</Select.Option>
                <Select.Option value={2}>多选</Select.Option>
              </Select>
            </Form.Item>

            <Form.Item name="max_choices" label="最多可选">
              <InputNumber min={1} max={100} />
            </Form.Item>

            <Form.Item name="max_votes" label="每人可投次数">
              <InputNumber min={1} max={100} />
            </Form.Item>
          </Space>

          <Form.Item name="time_range" label="投票时间范围">
            <DatePicker.RangePicker showTime format="YYYY-MM-DD HH:mm" style={{ width: '100%' }} />
          </Form.Item>

          <Form.Item name="status" label="立即开始" valuePropName="checked">
            <Switch />
          </Form.Item>

          <Divider>投票选项</Divider>

          <Form.List name="options" rules={[{ validator: async (_, value) => {
            if (!value || value.length < 1) {
              return Promise.reject(new Error('至少需要一个选项'));
            }
          }}]}>
            {(fields, { add, remove }, { errors }) => (
              <>
                {fields.map((field, index) => (
                  <Space key={field.key} align="baseline" style={{ display: 'flex', marginBottom: 8 }}>
                    <Form.Item
                      {...field}
                      name={[field.name, 'content']}
                      rules={[{ required: true, message: '请输入选项内容' }]}
                    >
                      <Input placeholder={`选项 ${index + 1}`} style={{ width: 300 }} maxLength={500} />
                    </Form.Item>
                    <Form.Item {...field} name={[field.name, 'image_url']}>
                      <Input placeholder="配图 URL(可选)" style={{ width: 200 }} />
                    </Form.Item>
                    {fields.length > 1 && (
                      <MinusCircleOutlined onClick={() => remove(field.name)} style={{ color: 'red' }} />
                    )}
                  </Space>
                ))}
                <Form.Item>
                  <Button type="dashed" onClick={() => add({ content: '', image_url: '' })} block icon={<PlusOutlined />}>
                    添加选项
                  </Button>
                  <Form.ErrorList errors={errors} />
                </Form.Item>
              </>
            )}
          </Form.List>

          <Form.Item>
            <Button type="primary" loading={loading} onClick={handleSubmit} size="large" block>
              创建投票
            </Button>
          </Form.Item>
        </Form>
      </Card>
    </div>
  );
}

export default VoteCreate;
