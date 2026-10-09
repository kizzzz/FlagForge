import type { Cause } from '../api/client'
export const causeText: Record<Cause, string> = {
  matched: '命中', attribute_missing: '属性缺失', value_not_equal: '值不相等', value_not_in_list: '不在列表中', type_mismatch: '类型不匹配', short_circuited_by_disabled_flag: '停用短路',
}
