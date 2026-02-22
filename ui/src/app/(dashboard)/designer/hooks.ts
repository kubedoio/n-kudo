'use client';

import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authStore } from '@/store/auth';
import type { Template, CreateTemplateRequest } from '@/api/types';
import type { DesignerTemplate } from '@/types/designer';

// Query keys for templates
export const templateKeys = {
    all: ['templates'] as const,
    lists: () => [...templateKeys.all, 'list'] as const,
    list: (filters: string) => [...templateKeys.lists(), { filters }] as const,
    details: () => [...templateKeys.all, 'detail'] as const,
    detail: (id: string) => [...templateKeys.details(), id] as const,
};

/**
 * Hook to fetch all templates
 */
export function useTemplates() {
    const client = authStore.getState().client;

    return useQuery({
        queryKey: templateKeys.lists(),
        queryFn: async () => {
            if (!client) throw new Error('API client not available');
            return client.listTemplates();
        },
        enabled: !!client,
    });
}

/**
 * Hook to fetch a single template by ID
 */
export function useTemplate(templateId: string) {
    const client = authStore.getState().client;

    return useQuery({
        queryKey: templateKeys.detail(templateId),
        queryFn: async () => {
            if (!client) throw new Error('API client not available');
            return client.getTemplate(templateId);
        },
        enabled: !!client && !!templateId,
    });
}

/**
 * Hook to create a new template
 */
export function useCreateTemplate() {
    const queryClient = useQueryClient();
    const client = authStore.getState().client;

    return useMutation({
        mutationFn: async (template: CreateTemplateRequest) => {
            if (!client) throw new Error('API client not available');
            return client.createTemplate(template);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
        },
    });
}

/**
 * Hook to update an existing template
 */
export function useUpdateTemplate() {
    const queryClient = useQueryClient();
    const client = authStore.getState().client;

    return useMutation({
        mutationFn: async ({ templateId, data }: { templateId: string; data: CreateTemplateRequest }) => {
            if (!client) throw new Error('API client not available');
            return client.updateTemplate(templateId, data);
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
            queryClient.invalidateQueries({ queryKey: templateKeys.detail(variables.templateId) });
        },
    });
}

/**
 * Hook to delete a template
 */
export function useDeleteTemplate() {
    const queryClient = useQueryClient();
    const client = authStore.getState().client;

    return useMutation({
        mutationFn: async (templateId: string) => {
            if (!client) throw new Error('API client not available');
            return client.deleteTemplate(templateId);
        },
        onSuccess: () => {
            queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
        },
    });
}

/**
 * Hook to save a template (create or update based on ID)
 */
export function useSaveTemplate() {
    const queryClient = useQueryClient();
    const client = authStore.getState().client;

    return useMutation({
        mutationFn: async ({
            templateId,
            data,
        }: {
            templateId?: string;
            data: CreateTemplateRequest;
        }) => {
            if (!client) throw new Error('API client not available');
            if (templateId) {
                return client.updateTemplate(templateId, data);
            }
            return client.createTemplate(data);
        },
        onSuccess: (_, variables) => {
            queryClient.invalidateQueries({ queryKey: templateKeys.lists() });
            if (variables.templateId) {
                queryClient.invalidateQueries({ queryKey: templateKeys.detail(variables.templateId) });
            }
        },
    });
}

/**
 * Hook for local template operations (download/upload JSON)
 */
export function useLocalTemplate() {
    /**
     * Download template as JSON file
     */
    const downloadTemplate = (template: DesignerTemplate | Template) => {
        const blob = new Blob([JSON.stringify(template, null, 2)], {
            type: 'application/json',
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = `${template.name.replace(/\s+/g, '_').toLowerCase()}.json`;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
    };

    /**
     * Upload/parse template from JSON file
     */
    const uploadTemplate = (file: File): Promise<Template | DesignerTemplate> => {
        return new Promise((resolve, reject) => {
            const reader = new FileReader();

            reader.onload = (e) => {
                try {
                    const content = e.target?.result as string;
                    const template = JSON.parse(content);
                    resolve(template);
                } catch (error) {
                    reject(new Error('Invalid template file format'));
                }
            };

            reader.onerror = () => {
                reject(new Error('Failed to read file'));
            };

            reader.readAsText(file);
        });
    };

    return { downloadTemplate, uploadTemplate };
}

// Import PlanAction type for deployment hook
import type { PlanAction } from './plan-generator';

/**
 * Hook for deploying plan actions directly to a site
 * Used by the designer for deploying canvas designs
 */
export function useDeployTemplateActions() {
    return useMutation({
        mutationFn: async ({
            siteId,
            actions,
        }: {
            siteId: string;
            actions: PlanAction[];
        }): Promise<{ plan: any; executions: any[] }> => {
            const client = authStore.getState().client;
            if (!client) {
                throw new Error('API client not available');
            }
            return client.deployTemplateToSite(siteId, actions);
        },
    });
}
