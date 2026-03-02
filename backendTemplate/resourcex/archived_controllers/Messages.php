<?php
class Messages extends Controller
{
    public function __construct()
    {
        $this->messageModel = $this->model('Message');
        $this->userModel = $this->model('User');
        $this->dashboardModel = $this->model('Dashboard');
        $this->serverKey = 'secret_server_key';
    }

    // Get user's message threads
    public function getThreads()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            try {
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $offset = ($page - 1) * $limit;

                $threads = $this->messageModel->getUserThreads($userData->user_id, $limit, $offset);

                $response = [
                    'status' => true,
                    'message' => 'Message threads retrieved successfully',
                    'data' => $threads,
                    'pagination' => [
                        'current_page' => $page,
                        'limit' => $limit
                    ]
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving threads: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Get messages in a thread
    public function getThreadMessages($thread_id = null)
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            if (!$thread_id) {
                $thread_id = $_GET['thread_id'] ?? '';
            }

            if (empty($thread_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Thread ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $page = isset($_GET['page']) ? (int)$_GET['page'] : 1;
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 50;
                $offset = ($page - 1) * $limit;

                $messages = $this->messageModel->getThreadMessages($thread_id, $userData->user_id, $limit, $offset);

                if ($messages === false) {
                    $response = [
                        'status' => false,
                        'message' => 'Unauthorized to view this thread or thread not found'
                    ];
                    print_r(json_encode($response));
                    exit;
                }

                // Mark messages as read
                $this->messageModel->markMessagesAsRead($thread_id, $userData->user_id);

                $response = [
                    'status' => true,
                    'message' => 'Thread messages retrieved successfully',
                    'data' => $messages,
                    'thread_id' => $thread_id
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving messages: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Send a message
    public function sendMessage()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST') {
            $postData = $this->getData();
            
            $receiver_id = trim($postData['receiver_id'] ?? '');
            $message_text = trim($postData['message_text'] ?? '');
            $subject = trim($postData['subject'] ?? '');
            $listing_id = trim($postData['listing_id'] ?? '');
            $message_type = trim($postData['message_type'] ?? 'general');

            // Validation
            if (empty($receiver_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Receiver ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            if (empty($message_text)) {
                $response = [
                    'status' => false,
                    'message' => 'Message text is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            // Check if receiver exists
            $receiver = $this->userModel->findUserById($receiver_id);
            if (!$receiver) {
                $response = [
                    'status' => false,
                    'message' => 'Receiver not found'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $message_id = $this->messageModel->sendMessage(
                    $userData->user_id,
                    $receiver_id,
                    $message_text,
                    $subject ?: null,
                    $listing_id ?: null,
                    $message_type
                );

                if ($message_id) {
                    // Track activity
                    $this->dashboardModel->trackActivity($userData->user_id, 'message_send', $message_id);

                    $response = [
                        'status' => true,
                        'message' => 'Message sent successfully',
                        'message_id' => $message_id
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to send message'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error sending message: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Get unread message count
    public function getUnreadCount()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            try {
                $unreadCount = $this->messageModel->getUnreadCount($userData->user_id);

                $response = [
                    'status' => true,
                    'message' => 'Unread count retrieved successfully',
                    'unread_count' => $unreadCount
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error retrieving unread count: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Search messages
    public function searchMessages()
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'GET') {
            $searchTerm = isset($_GET['search']) ? trim($_GET['search']) : '';
            
            if (empty($searchTerm)) {
                $response = [
                    'status' => false,
                    'message' => 'Search term is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                $limit = isset($_GET['limit']) ? (int)$_GET['limit'] : 20;
                $messages = $this->messageModel->searchMessages($userData->user_id, $searchTerm, $limit);

                $response = [
                    'status' => true,
                    'message' => 'Message search completed successfully',
                    'data' => $messages,
                    'search_term' => $searchTerm,
                    'count' => count($messages)
                ];

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error searching messages: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }

    // Delete a message
    public function deleteMessage($message_id = null)
    {
        try {
            $userData = $this->RouteProtection();
        } catch (UnexpectedValueException $e) {
            $res = [
                'status' => 401,
                'message' => $e->getMessage(),
            ];
            print_r(json_encode($res));
            exit;
        }

        if ($_SERVER['REQUEST_METHOD'] == 'POST' || $_SERVER['REQUEST_METHOD'] == 'DELETE') {
            if (!$message_id) {
                $postData = $this->getData();
                $message_id = trim($postData['message_id'] ?? '');
            }

            if (empty($message_id)) {
                $response = [
                    'status' => false,
                    'message' => 'Message ID is required'
                ];
                print_r(json_encode($response));
                exit;
            }

            try {
                if ($this->messageModel->deleteMessage($message_id, $userData->user_id)) {
                    $response = [
                        'status' => true,
                        'message' => 'Message deleted successfully'
                    ];
                } else {
                    $response = [
                        'status' => false,
                        'message' => 'Failed to delete message or message not found'
                    ];
                }

                print_r(json_encode($response));
                exit;

            } catch (Exception $e) {
                $response = [
                    'status' => false,
                    'message' => 'Error deleting message: ' . $e->getMessage()
                ];
                print_r(json_encode($response));
                exit;
            }
        } else {
            $response = [
                'status' => false,
                'message' => 'Invalid request method'
            ];
            print_r(json_encode($response));
            exit;
        }
    }
}
