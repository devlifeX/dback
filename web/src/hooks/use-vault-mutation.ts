import {
  useMutation,
  useQueryClient,
  type UseMutationOptions,
  type UseMutationResult,
} from '@tanstack/react-query'
import { toast } from 'sonner'
import { ApiClientError, fetchRevision } from '@/api/client'

type VaultMutationFn<TData, TVariables> = (variables: TVariables, etag: string) => Promise<TData>

type VaultMutationOptions<TData, TVariables> = Omit<
  UseMutationOptions<TData, Error, TVariables>,
  'mutationFn'
> & {
  mutationFn: VaultMutationFn<TData, TVariables>
  invalidateKeys?: readonly (readonly string[])[]
}

export function useMutationWithRevision<TData, TVariables>(
  options: VaultMutationOptions<TData, TVariables>,
): UseMutationResult<TData, Error, TVariables> {
  const qc = useQueryClient()
  const { mutationFn, invalidateKeys, onError, onSuccess, ...rest } = options

  return useMutation({
    ...rest,
    mutationFn: async (variables: TVariables) => {
      const { etag } = await fetchRevision()
      return mutationFn(variables, etag)
    },
    onError: (error, variables, onMutateResult, context) => {
      if (error instanceof ApiClientError && error.status === 412) {
        toast.error('Data changed elsewhere. Refresh and try again.')
        void qc.invalidateQueries({ queryKey: ['revision'] })
      } else {
        toast.error(error.message)
      }
      onError?.(error, variables, onMutateResult, context)
    },
    onSuccess: (data, variables, onMutateResult, context) => {
      if (invalidateKeys) {
        for (const key of invalidateKeys) {
          void qc.invalidateQueries({ queryKey: key })
        }
      }
      void qc.invalidateQueries({ queryKey: ['revision'] })
      onSuccess?.(data, variables, onMutateResult, context)
    },
  })
}
