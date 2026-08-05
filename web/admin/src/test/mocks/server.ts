// web/admin/src/test/mocks/server.ts
// MSW Server 配置

import { setupServer } from 'msw/node';
import { handlers } from './handlers';

export const server = setupServer(...handlers);
